package hikvision

import (
	"encoding/json"
	"fmt"
	"time"
)

// Access-control event minor codes, confirmed against a DS-K1T8003MF by
// scanning a valid member and an expired one and reading the log back.
// Hikvision publishes the full table only through its partner portal, so
// anything not listed here is treated as noise (door buttons, tamper, relay
// activity) and logged rather than guessed at.
const (
	// MinorFingerprintGranted is a successful fingerprint entry.
	MinorFingerprintGranted = 38
	// MinorValidityExpired is the terminal refusing somebody it recognised
	// because their validity window has closed — which, since gym-go drives
	// that window from the subscription, means "membership not current".
	MinorValidityExpired = 8
	// MinorOutsideSchedule is the refusal of somebody recognised and within
	// their validity window, but outside the hours of their plan template.
	//
	// NOT yet confirmed on the DS-K1T8003MF: it is Hikvision's documented
	// "invalid period" code, the neighbour of the confirmed 8 in the same
	// family. Confirm it with TestLiveAccessSchedule by scanning outside the
	// test schedule; if the device answers another code, change it here.
	MinorOutsideSchedule = 7
)

// AcsEvent is one row of the terminal's access log.
type AcsEvent struct {
	// SerialNo increases with every event and is the ingestion cursor.
	SerialNo int
	Major    int
	Minor    int
	At       time.Time
	// EmployeeNo is empty for events not tied to a person (door buttons,
	// remote opens, tamper alarms).
	EmployeeNo string
	DoorNo     int
}

// Granted reports whether the door actually opened for this event.
func (e AcsEvent) Granted() bool { return e.Minor == MinorFingerprintGranted }

// DeniedForExpiry reports the "recognised, but not current" refusal.
func (e AcsEvent) DeniedForExpiry() bool { return e.Minor == MinorValidityExpired }

// DeniedForSchedule reports the "recognised, but not at this hour" refusal.
func (e AcsEvent) DeniedForSchedule() bool { return e.Minor == MinorOutsideSchedule }

// AboutAPerson filters out the events that carry no identity.
func (e AcsEvent) AboutAPerson() bool { return e.EmployeeNo != "" }

type acsEventResponse struct {
	AcsEvent struct {
		NumOfMatches   int    `json:"numOfMatches"`
		TotalMatches   int    `json:"totalMatches"`
		ResponseStatus string `json:"responseStatusStrg"`
		InfoList       []struct {
			Major      int    `json:"major"`
			Minor      int    `json:"minor"`
			Time       string `json:"time"`
			EmployeeNo string `json:"employeeNoString"`
			DoorNo     int    `json:"doorNo"`
			SerialNo   int    `json:"serialNo"`
		} `json:"InfoList"`
	} `json:"AcsEvent"`
}

// SearchEvents reads a page of the access log between two instants.
//
// The device pages with an absolute offset, so callers walk forward with
// `position`; `total` is how many events match the range in all.
//
// Timestamps come back as RFC 3339 with the device's configured offset —
// which makes them only as correct as the terminal's timezone. A unit fresh
// out of the box is set to CST-8 (China) and will stamp every entry 13 hours
// off, so the device's clock must be set to the gym's timezone before any of
// this is trustworthy.
func (c *Client) SearchEvents(from, to time.Time, position, maxResults int) (events []AcsEvent, total int, err error) {
	body, err := json.Marshal(map[string]any{
		"AcsEventCond": map[string]any{
			"searchID":             "gym-go",
			"searchResultPosition": position,
			"maxResults":           maxResults,
			"major":                0,
			"minor":                0,
			"startTime":            from.Format(time.RFC3339),
			"endTime":              to.Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, 0, err
	}

	respBody, status, err := c.do("POST", "/ISAPI/AccessControl/AcsEvent?format=json", "application/json", body)
	if err != nil {
		return nil, 0, err
	}
	if status < 200 || status >= 300 {
		return nil, 0, fmt.Errorf("search events: device returned HTTP %s: %s", httpStatusText(status), string(respBody))
	}

	var parsed acsEventResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, 0, fmt.Errorf("search events: unexpected response: %s", string(respBody))
	}

	for _, row := range parsed.AcsEvent.InfoList {
		at, err := time.Parse(time.RFC3339, row.Time)
		if err != nil {
			// Keep the event: a timestamp we cannot read is worth less than
			// losing the entry entirely, and the caller falls back to now.
			at = time.Time{}
		}
		events = append(events, AcsEvent{
			SerialNo:   row.SerialNo,
			Major:      row.Major,
			Minor:      row.Minor,
			At:         at,
			EmployeeNo: row.EmployeeNo,
			DoorNo:     row.DoorNo,
		})
	}
	return events, parsed.AcsEvent.TotalMatches, nil
}

// CountEvents returns how many events the terminal holds for a range,
// without pulling them.
func (c *Client) CountEvents(from, to time.Time) (int, error) {
	_, total, err := c.SearchEvents(from, to, 0, 1)
	return total, err
}
