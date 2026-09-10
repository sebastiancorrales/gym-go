package entities

import (
	"time"

	"github.com/google/uuid"
)

// AccessEventCursor remembers how far a terminal's event log has been
// ingested, so a restart does not re-import entries as duplicate check-ins.
//
// It is its own table rather than a couple of columns on Device on purpose:
// the ingester writes it every few seconds while an administrator may be
// saving the device's configuration at the same time, and DeviceRepository
// updates the whole row — the two would silently overwrite each other.
type AccessEventCursor struct {
	// DeviceID is the terminal this cursor belongs to.
	DeviceID uuid.UUID `json:"device_id" gorm:"primaryKey;type:text"`
	// LastSerialNo is the highest event serial already turned into an access
	// log. The device numbers events monotonically, which makes this an
	// exact "everything up to here is done" marker.
	LastSerialNo int `json:"last_serial_no"`
	// LastEventAt is when that event happened, kept for diagnostics and to
	// bound the next query's time range.
	LastEventAt time.Time `json:"last_event_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AccessEventCursor) TableName() string {
	return "access_event_cursors"
}
