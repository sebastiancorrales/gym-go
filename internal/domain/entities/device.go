package entities

import (
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
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// NewDevice creates a new device
func NewDevice(gymID uuid.UUID, name string, deviceType DeviceType, serialNumber, location string) *Device {
	now := time.Now()
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
	now := time.Now()
	d.Status = DeviceStatusOnline
	d.LastHeartbeat = &now
	d.UpdatedAt = now
}

// SetOffline marks device as offline
func (d *Device) SetOffline() {
	d.Status = DeviceStatusOffline
	d.UpdatedAt = time.Now()
}

// SetMaintenance sets device to maintenance mode
func (d *Device) SetMaintenance() {
	d.Status = DeviceStatusMaintenance
	d.UpdatedAt = time.Now()
}

// SetError marks device with error
func (d *Device) SetError() {
	d.Status = DeviceStatusError
	d.UpdatedAt = time.Now()
}

// Heartbeat updates last heartbeat
func (d *Device) Heartbeat() {
	now := time.Now()
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



