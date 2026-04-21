package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/domain/repositories"
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
	c.JSON(http.StatusOK, devices)
}

// Create creates a new relay device
func (h *DeviceHandler) Create(c *gin.Context) {
	gymID, ok := gymIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "gym not found in context"})
		return
	}

	var req struct {
		Name     string `json:"name" binding:"required"`
		Location string `json:"location"`
		COMPort  string `json:"com_port"`
		BaudRate int    `json:"baud_rate"`
		Notes    string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	baudRate := req.BaudRate
	if baudRate == 0 {
		baudRate = 9600
	}

	device := entities.NewDevice(gymID, req.Name, entities.DeviceTypeRelay, "", req.Location)
	device.COMPort = req.COMPort
	device.BaudRate = baudRate
	device.Notes = req.Notes

	if err := h.repo.Create(device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
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

	var req struct {
		Name     string `json:"name"`
		Location string `json:"location"`
		COMPort  string `json:"com_port"`
		BaudRate int    `json:"baud_rate"`
		Notes    string `json:"notes"`
		IsActive *bool  `json:"is_active"`
	}
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
	if req.COMPort != "" {
		device.COMPort = req.COMPort
	}
	if req.BaudRate != 0 {
		device.BaudRate = req.BaudRate
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

// Trigger sends "OPEN\n" to the device's configured COM port
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
