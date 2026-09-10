package entities

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// DeviceType represents device type
type DeviceType string

const (
	DeviceTypeTurnstile   DeviceType = "TURNSTILE"
	DeviceTypeFingerprint DeviceType = "FINGERPRINT"
	DeviceTypeFaceID      DeviceType = "FACE_ID"
	DeviceTypeCamera      DeviceType = "CAMERA"
	DeviceTypeTablet      DeviceType = "TABLET"
	DeviceTypeKiosk       DeviceType = "KIOSK"
	DeviceTypeRelay       DeviceType = "RELAY"
	// DeviceTypeHikvisionTerminal is a standalone access-control terminal
	// (e.g. DS-K1T8003MF) reached over the network via Hikvision's ISAPI —
	// as opposed to DeviceTypeRelay, which is a serial relay board wired to
	// a COM port. Its connection details live in Configuration (see
	// HikvisionConfig), not in COMPort/BaudRate.
	DeviceTypeHikvisionTerminal DeviceType = "HIKVISION_TERMINAL"
)

// DeviceStatus represents device status
type DeviceStatus string

const (
	DeviceStatusOnline      DeviceStatus = "ONLINE"
	DeviceStatusOffline     DeviceStatus = "OFFLINE"
	DeviceStatusMaintenance DeviceStatus = "MAINTENANCE"
	DeviceStatusError       DeviceStatus = "ERROR"
)

// Device represents a hardware device for access control
type Device struct {
	ID              uuid.UUID    `json:"id"`
	GymID           uuid.UUID    `json:"gym_id"`
	Name            string       `json:"name"`
	DeviceType      DeviceType   `json:"device_type"`
	SerialNumber    string       `json:"serial_number"`
	IPAddress       string       `json:"ip_address,omitempty"`
	MACAddress      string       `json:"mac_address,omitempty"`
	Location        string       `json:"location"`
	Status          DeviceStatus `json:"status"`
	LastHeartbeat   *time.Time   `json:"last_heartbeat,omitempty"`
	FirmwareVersion string       `json:"firmware_version,omitempty"`
	Configuration   string       `json:"configuration,omitempty"`
	IsActive        bool         `json:"is_active"`
	Notes           string       `json:"notes,omitempty"`
	COMPort         string       `json:"com_port,omitempty"`
	BaudRate        int          `json:"baud_rate,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// NewDevice creates a new device
func NewDevice(gymID uuid.UUID, name string, deviceType DeviceType, serialNumber, location string) *Device {
	now := time.Now().UTC().Round(0)
	return &Device{
		ID:           uuid.New(),
		GymID:        gymID,
		Name:         name,
		DeviceType:   deviceType,
		SerialNumber: serialNumber,
		Location:     location,
		Status:       DeviceStatusOffline,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetOnline marks device as online
func (d *Device) SetOnline() {
	now := time.Now().UTC().Round(0)
	d.Status = DeviceStatusOnline
	d.LastHeartbeat = &now
	d.UpdatedAt = now
}

// SetOffline marks device as offline
func (d *Device) SetOffline() {
	d.Status = DeviceStatusOffline
	d.UpdatedAt = time.Now().UTC().Round(0)
}

// SetMaintenance sets device to maintenance mode
func (d *Device) SetMaintenance() {
	d.Status = DeviceStatusMaintenance
	d.UpdatedAt = time.Now().UTC().Round(0)
}

// SetError marks device with error
func (d *Device) SetError() {
	d.Status = DeviceStatusError
	d.UpdatedAt = time.Now().UTC().Round(0)
}

// Heartbeat updates last heartbeat
func (d *Device) Heartbeat() {
	now := time.Now().UTC().Round(0)
	d.LastHeartbeat = &now
	if d.Status == DeviceStatusOffline {
		d.Status = DeviceStatusOnline
	}
	d.UpdatedAt = now
}

// IsOnline checks if device is online
func (d *Device) IsOnline() bool {
	return d.Status == DeviceStatusOnline && d.IsActive
}

// HikvisionConfig is the shape stored (as JSON) in Device.Configuration for a
// DeviceTypeHikvisionTerminal. The IP itself stays in Device.IPAddress —
// that field already existed for this purpose — this struct only adds what
// IPAddress can't carry: the HTTP port, the admin credentials ISAPI needs
// for Digest auth, and which door number to command (single-door terminals
// like the DS-K1T8003MF use door 1).
type HikvisionConfig struct {
	HTTPPort int    `json:"http_port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DoorNo   int    `json:"door_no"`
}

// ErrNotHikvisionTerminal is returned by ParseHikvisionConfig when called on
// a device that isn't a DeviceTypeHikvisionTerminal.
var ErrNotHikvisionTerminal = errors.New("device is not a Hikvision terminal")

// ParseHikvisionConfig decodes Device.Configuration into a HikvisionConfig,
// filling in the documented defaults (HTTP port 80, door 1) when the field
// is left blank in the request.
func (d *Device) ParseHikvisionConfig() (HikvisionConfig, error) {
	if d.DeviceType != DeviceTypeHikvisionTerminal {
		return HikvisionConfig{}, ErrNotHikvisionTerminal
	}
	cfg := HikvisionConfig{HTTPPort: 80, DoorNo: 1}
	if d.Configuration != "" {
		if err := json.Unmarshal([]byte(d.Configuration), &cfg); err != nil {
			return HikvisionConfig{}, err
		}
	}
	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = 80
	}
	if cfg.DoorNo == 0 {
		cfg.DoorNo = 1
	}
	return cfg, nil
}



