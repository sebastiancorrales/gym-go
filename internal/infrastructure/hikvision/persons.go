package hikvision

import (
	"encoding/json"
	"fmt"
	"time"
)

// Person is a person record on the terminal. employeeNo is the device's
// primary key for a person and is capped at 32 characters, which is exactly
// the length of a gym-go UUID with its hyphens stripped — see
// EmployeeNoForUser in the use case.
type Person struct {
	EmployeeNo string
	Name       string
	ValidFrom  time.Time
	ValidTo    time.Time
	// PlanTemplates lists the plan templates that decide the hours this
	// person may enter, comma-separated as the device writes them ("1", "3",
	// "3,4"). Empty means the all-day template 1. See schedules.go.
	PlanTemplates string
}

// AllDayTemplate is the device's built-in template with no hour restriction.
const AllDayTemplate = "1"

func (p Person) planTemplates() string {
	if p.PlanTemplates == "" {
		return AllDayTemplate
	}
	return p.PlanTemplates
}

// isapiStatus is the plain ack most JSON ISAPI calls return. statusCode 1 is
// the device's success code.
type isapiStatus struct {
	StatusCode    int    `json:"statusCode"`
	StatusString  string `json:"statusString"`
	SubStatusCode string `json:"subStatusCode"`
}

func (s isapiStatus) err(op string) error {
	if s.StatusCode == 1 {
		return nil
	}
	return &DeviceError{Op: op, StatusString: s.StatusString, SubStatusCode: s.SubStatusCode}
}

// deviceTimeLayout is how ISAPI expects local timestamps: no zone suffix.
const deviceTimeLayout = "2006-01-02T15:04:05"

// maxDeviceTime is the latest instant the terminal accepts. It keeps time in
// a signed 32-bit counter, so anything from 2038-01-01 onwards is refused
// with "timeFormatError" — and, when it arrives via Modify, with the far less
// helpful "badParameters". Validity windows are clamped rather than rejected:
// a membership that runs past 2037 is not a reason to fail an enrollment.
//
// The limit is the device.s own local wall clock, not UTC: it is compared
// against values that are converted to local before being sent.
var maxDeviceTime = time.Date(2037, time.December, 31, 23, 59, 59, 0, time.Local)

func clampToDeviceRange(t time.Time) time.Time {
	if t.After(maxDeviceTime) {
		return maxDeviceTime
	}
	return t
}

// deviceLocalTime renders an instant the way the terminal stores it:
// "local" wall-clock with no zone suffix.
//
// The conversion to local matters. Subscription dates arrive in whatever zone
// the database handed back — often UTC — and formatting one of those directly
// sends its UTC wall clock as if it were gym time, shifting the window by the
// UTC offset (five hours in Bogotá). Beyond being wrong, it made the window
// read back different from the one written, so the sync rewrote the same
// member on every pass forever.
func deviceLocalTime(t time.Time) string {
	return clampToDeviceRange(t).In(time.Local).Format(deviceTimeLayout)
}

// personPayload is the UserInfo body shared by Record (create) and Modify.
//
// doorRight and RightPlan are the part that is easy to miss: a person created
// without them is stored fine and their fingerprint enrolls fine, but every
// scan is denied at the door — the terminal reads the finger and then refuses
// because the person has no access schedule. planTemplateNo "1" is the
// device's built-in all-day template; a member of a plan with restricted
// hours carries that plan's template instead.
func personPayload(p Person) map[string]any {
	return map[string]any{
		"UserInfo": map[string]any{
			"employeeNo": p.EmployeeNo,
			"name":       p.Name,
			"userType":   "normal",
			"Valid": map[string]any{
				"enable":    true,
				"beginTime": deviceLocalTime(p.ValidFrom),
				"endTime":   deviceLocalTime(p.ValidTo),
				"timeType":  "local",
			},
			"doorRight": "1",
			"RightPlan": []map[string]any{
				{"doorNo": 1, "planTemplateNo": p.planTemplates()},
			},
		},
	}
}

// CreatePerson adds a person to the terminal.
func (c *Client) CreatePerson(p Person) error {
	body, err := json.Marshal(personPayload(p))
	if err != nil {
		return err
	}
	return c.jsonStatusCall("POST", "/ISAPI/AccessControl/UserInfo/Record?format=json", body, "create person")
}

// ModifyPerson updates an existing person (same payload as create).
func (c *Client) ModifyPerson(p Person) error {
	body, err := json.Marshal(personPayload(p))
	if err != nil {
		return err
	}
	return c.jsonStatusCall("PUT", "/ISAPI/AccessControl/UserInfo/Modify?format=json", body, "modify person")
}

// UpsertPerson creates the person, falling back to a modify when the device
// says the employeeNo already exists. The device has no "create or replace",
// and re-enrolling an existing member is the common case.
//
// Both failures are reported. When a payload is bad for some other reason the
// two calls fail differently — create answered "timeFormatError" for an
// out-of-range validity date while modify answered only "badParameters" —
// and returning just the second one hides the diagnosis.
func (c *Client) UpsertPerson(p Person) error {
	createErr := c.CreatePerson(p)
	if createErr == nil {
		return nil
	}

	modifyErr := c.ModifyPerson(p)
	if modifyErr == nil {
		return nil
	}
	return fmt.Errorf("could not create (%v) nor update (%v) the person on the device", createErr, modifyErr)
}

// DeletePerson removes a person from the terminal. Their fingerprints go with
// them, but DeleteFingerprints is still the way to free a finger for
// re-enrollment under a different person — see the duplicate rule there.
func (c *Client) DeletePerson(employeeNo string) error {
	body, err := json.Marshal(map[string]any{
		"UserInfoDelCond": map[string]any{
			"EmployeeNoList": []map[string]string{{"employeeNo": employeeNo}},
		},
	})
	if err != nil {
		return err
	}
	return c.jsonStatusCall("PUT", "/ISAPI/AccessControl/UserInfo/Delete?format=json", body, "delete person")
}

// FindPerson returns the person with this employeeNo, or nil when the device
// has no such record.
func (c *Client) FindPerson(employeeNo string) (*Person, error) {
	people, _, err := c.searchPersons(map[string]any{
		"searchID":             "gym-go",
		"searchResultPosition": 0,
		"maxResults":           1,
		"EmployeeNoList":       []map[string]string{{"employeeNo": employeeNo}},
	})
	if err != nil || len(people) == 0 {
		return nil, err
	}
	return &people[0], nil
}

// personPageSize is what each search asks for. The terminal ignores it above
// its own limit and answers at most ten people per request, which is why
// paging must follow the device's totalMatches and never "I got fewer than I
// asked for, so that was the last page" — that read stops at person ten and
// silently reports everybody after them as missing from the device.
const personPageSize = 10

// maxPersonPages bounds the walk so a device that keeps answering "there are
// more" cannot spin forever. It covers the model's 1000-person capacity.
const maxPersonPages = 120

// ListPersons reads every person the terminal holds, with their validity
// window.
//
// The sync uses it to compare before writing. Reading all of them costs one
// request per ten people; writing all of them costs one request each, hits
// the device's embedded database, and is what collides with the event poller
// and produces updateDatabaseError. Reads are cheap and safe, writes are
// neither.
func (c *Client) ListPersons() ([]Person, error) {
	var all []Person
	for page := 0; page < maxPersonPages; page++ {
		people, total, err := c.searchPersons(map[string]any{
			"searchID":             "gym-go",
			"searchResultPosition": len(all),
			"maxResults":           personPageSize,
		})
		if err != nil {
			return nil, err
		}
		if len(people) == 0 {
			return all, nil
		}

		all = append(all, people...)
		if len(all) >= total {
			return all, nil
		}
	}
	return all, fmt.Errorf("list persons: the device kept reporting more after %d pages", maxPersonPages)
}

// searchPersons runs one UserInfo/Search, returning its people and how many
// the device says match in total.
func (c *Client) searchPersons(cond map[string]any) ([]Person, int, error) {
	body, err := json.Marshal(map[string]any{"UserInfoSearchCond": cond})
	if err != nil {
		return nil, 0, err
	}

	respBody, status, err := c.do("POST", "/ISAPI/AccessControl/UserInfo/Search?format=json", "application/json", body)
	if err != nil {
		return nil, 0, err
	}
	if status < 200 || status >= 300 {
		return nil, 0, fmt.Errorf("search person: device returned HTTP %s: %s", httpStatusText(status), string(respBody))
	}

	var parsed struct {
		UserInfoSearch struct {
			NumOfMatches int `json:"numOfMatches"`
			TotalMatches int `json:"totalMatches"`
			UserInfo     []struct {
				EmployeeNo string `json:"employeeNo"`
				Name       string `json:"name"`
				Valid      struct {
					Enable    bool   `json:"enable"`
					BeginTime string `json:"beginTime"`
					EndTime   string `json:"endTime"`
				} `json:"Valid"`
				RightPlan []struct {
					DoorNo         int    `json:"doorNo"`
					PlanTemplateNo string `json:"planTemplateNo"`
				} `json:"RightPlan"`
			} `json:"UserInfo"`
		} `json:"UserInfoSearch"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, 0, fmt.Errorf("search person: unexpected response: %s", string(respBody))
	}

	people := make([]Person, 0, len(parsed.UserInfoSearch.UserInfo))
	for _, row := range parsed.UserInfoSearch.UserInfo {
		person := Person{
			EmployeeNo: row.EmployeeNo,
			Name:       row.Name,
			ValidFrom:  parseDeviceTime(row.Valid.BeginTime),
			ValidTo:    parseDeviceTime(row.Valid.EndTime),
		}
		for _, right := range row.RightPlan {
			if right.DoorNo == 1 {
				person.PlanTemplates = right.PlanTemplateNo
			}
		}
		people = append(people, person)
	}
	return people, parsed.UserInfoSearch.TotalMatches, nil
}

// parseDeviceTime reads a timestamp the device wrote back. It answers with an
// offset ("2026-10-06T12:00:00-05:00") even though writes are sent without
// one, so both shapes are accepted; an unreadable value yields the zero time,
// which the sync treats as "different" and therefore rewrites.
func parseDeviceTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t
	}
	if t, err := time.ParseInLocation(deviceTimeLayout, raw, time.Local); err == nil {
		return t
	}
	return time.Time{}
}

// SameWindowAs reports whether the device already holds this exact validity
// window, name and hour template, to the second. Comparison is by wall-clock
// instant so a value read back with an offset still matches the one written
// without.
//
// p is what the device answered. When it did not report a template at all,
// the template is not compared: treating "not reported" as "different" would
// rewrite every member on every pass, which is precisely the write burst the
// comparison exists to prevent. Template changes then still travel with the
// write-through on the subscription change that causes them.
func (p Person) SameWindowAs(other Person) bool {
	if p.PlanTemplates != "" && p.PlanTemplates != other.planTemplates() {
		return false
	}
	return p.Name == other.Name &&
		p.ValidFrom.Truncate(time.Second).Equal(clampToDeviceRange(other.ValidFrom).Truncate(time.Second)) &&
		p.ValidTo.Truncate(time.Second).Equal(clampToDeviceRange(other.ValidTo).Truncate(time.Second))
}

// jsonStatusCall performs a JSON call whose response is a statusCode ack,
// retrying the refusals that mean "the device was busy just then".
//
// The device answers HTTP 400 for those, with the reason only in the JSON
// body, so the body is parsed before the status code is judged — reading it
// as a transport failure is what hid updateDatabaseError behind a wall of
// raw response text in the log.
func (c *Client) jsonStatusCall(method, path string, body []byte, op string) error {
	return withRetry(func() error {
		respBody, status, err := c.do(method, path, "application/json", body)
		if err != nil {
			return err
		}

		var parsed isapiStatus
		if jsonErr := json.Unmarshal(respBody, &parsed); jsonErr == nil && parsed.StatusCode != 0 {
			return parsed.err(op)
		}
		if status < 200 || status >= 300 {
			return fmt.Errorf("%s: device returned HTTP %s: %s", op, httpStatusText(status), string(respBody))
		}
		return fmt.Errorf("%s: unexpected response: %s", op, string(respBody))
	})
}
