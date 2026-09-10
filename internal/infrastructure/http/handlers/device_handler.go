package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/hikvision"
	"go.bug.st/serial"
)

type DeviceHandler struct {
	repo repositories.DeviceRepository
}

func NewDeviceHandler(repo repositories.DeviceRepository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

func gymIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("gym_id")
	if !exists {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw.(string))
	return id, err == nil
}

// List returns all relay devices for the gym
func (h *DeviceHandler) List(c *gin.Context) {
	gymID, ok := gymIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "gym not found in context"})
		return
	}
	devices, err := h.repo.FindByGymID(gymID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, d := range devices {
		redactDeviceSecret(d)
	}
	c.JSON(http.StatusOK, devices)
}

// redactDeviceSecret blanks the admin password inside Configuration before a
// device row goes out over JSON. This route is already PermDevices-only
// (ADMIN_GYM/SUPER_ADMIN), but there's no reason to echo the ISAPI
// credential back on every list/create/update response, or let it sit in a
// browser's network log — the frontend never needs to read it back, only to
// send a new one when the admin changes it.
func redactDeviceSecret(d *entities.Device) {
	if d.DeviceType != entities.DeviceTypeHikvisionTerminal || d.Configuration == "" {
		return
	}
	cfg, err := d.ParseHikvisionConfig()
	if err != nil {
		return
	}
	cfg.Password = ""
	raw, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	d.Configuration = string(raw)
}

// deviceRequest covers both device kinds the panel manages: a serial relay
// (com_port/baud_rate) and a Hikvision ISAPI terminal (ip_address +
// hikvision, which mirrors entities.HikvisionConfig). device_type selects
// which set of fields applies; it defaults to RELAY so existing callers
// that never send it keep behaving exactly as before.
type deviceRequest struct {
	Name       string                    `json:"name"`
	DeviceType entities.DeviceType       `json:"device_type"`
	Location   string                    `json:"location"`
	COMPort    string                    `json:"com_port"`
	BaudRate   int                       `json:"baud_rate"`
	IPAddress  string                    `json:"ip_address"`
	Hikvision  *entities.HikvisionConfig `json:"hikvision"`
	Notes      string                    `json:"notes"`
	IsActive   *bool                     `json:"is_active"`
}

// applyConnectionFields copies the fields relevant to device.DeviceType from
// req onto device — the two kinds of device don't share a connection shape,
// so this is where that split is resolved once instead of in every handler.
func applyConnectionFields(device *entities.Device, req deviceRequest) error {
	switch device.DeviceType {
	case entities.DeviceTypeHikvisionTerminal:
		if req.IPAddress != "" {
			device.IPAddress = req.IPAddress
		}
		if req.Hikvision != nil {
			// Merge onto the existing config rather than replacing it wholesale:
			// redactDeviceSecret strips the password before a device row ever
			// reaches the frontend, so a save that only renames the device (or
			// changes some other field) arrives here with password == "" and
			// must not blank out what's already stored.
			cfg, _ := device.ParseHikvisionConfig()
			if req.Hikvision.HTTPPort != 0 {
				cfg.HTTPPort = req.Hikvision.HTTPPort
			}
			if req.Hikvision.Username != "" {
				cfg.Username = req.Hikvision.Username
			}
			if req.Hikvision.Password != "" {
				cfg.Password = req.Hikvision.Password
			}
			if req.Hikvision.DoorNo != 0 {
				cfg.DoorNo = req.Hikvision.DoorNo
			}
			raw, err := json.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("invalid hikvision config: %w", err)
			}
			device.Configuration = string(raw)
		}
	default:
		if req.COMPort != "" {
			device.COMPort = req.COMPort
		}
		if req.BaudRate != 0 {
			device.BaudRate = req.BaudRate
		}
	}
	return nil
}

// Create registers a new device — a serial relay or a Hikvision terminal,
// per device_type.
func (h *DeviceHandler) Create(c *gin.Context) {
	gymID, ok := gymIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "gym not found in context"})
		return
	}

	var req deviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	deviceType := req.DeviceType
	if deviceType == "" {
		deviceType = entities.DeviceTypeRelay
	}
	if deviceType == entities.DeviceTypeRelay && req.BaudRate == 0 {
		req.BaudRate = 9600
	}

	device := entities.NewDevice(gymID, req.Name, deviceType, "", req.Location)
	device.Notes = req.Notes
	if err := applyConnectionFields(device, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	redactDeviceSecret(device)
	c.JSON(http.StatusCreated, device)
}

// Update updates a relay device
func (h *DeviceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	device, err := h.repo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	var req deviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		device.Name = req.Name
	}
	if req.Location != "" {
		device.Location = req.Location
	}
	// device_type is fixed at creation — switching a row between a serial
	// relay and a network terminal mid-life would leave its Configuration
	// describing the wrong shape. Delete and recreate it instead.
	if err := applyConnectionFields(device, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Notes != "" {
		device.Notes = req.Notes
	}
	if req.IsActive != nil {
		device.IsActive = *req.IsActive
	}
	device.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	redactDeviceSecret(device)
	c.JSON(http.StatusOK, device)
}

// Delete removes a relay device
func (h *DeviceHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "device deleted"})
}

// Trigger opens the device: a serial relay gets "OPEN\n" on its COM port, a
// Hikvision terminal gets the ISAPI remote-open command for its door.
func (h *DeviceHandler) Trigger(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	device, err := h.repo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	if !device.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device is inactive"})
		return
	}

	if device.DeviceType == entities.DeviceTypeHikvisionTerminal {
		h.triggerHikvision(c, device)
		return
	}
	h.triggerRelay(c, device)
}

func (h *DeviceHandler) triggerRelay(c *gin.Context, device *entities.Device) {
	if device.COMPort == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device has no COM port configured"})
		return
	}

	baudRate := device.BaudRate
	if baudRate == 0 {
		baudRate = 9600
	}

	mode := &serial.Mode{BaudRate: baudRate}
	port, err := serial.Open(device.COMPort, mode)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("cannot open %s: %v", device.COMPort, err)})
		return
	}
	defer port.Close()

	if _, err := port.Write([]byte("OPEN\n")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("write failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "triggered", "device": device.Name, "port": device.COMPort})
}

func (h *DeviceHandler) triggerHikvision(c *gin.Context, device *entities.Device) {
	if device.IPAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device has no IP address configured"})
		return
	}

	cfg, err := device.ParseHikvisionConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("invalid device configuration: %v", err)})
		return
	}
	if cfg.Username == "" || cfg.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device has no admin credentials configured"})
		return
	}

	client := hikvision.NewClient(device.IPAddress, cfg.HTTPPort, cfg.Username, cfg.Password)
	if err := client.OpenDoor(cfg.DoorNo); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("cannot open door: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "triggered", "device": device.Name, "ip": device.IPAddress})
}
