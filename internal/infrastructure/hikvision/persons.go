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
	return fmt.Errorf("%s failed: %s (%s)", op, s.StatusString, s.SubStatusCode)
}

// deviceTimeLayout is how ISAPI expects local timestamps: no zone suffix.
const deviceTimeLayout = "2006-01-02T15:04:05"

// maxDeviceTime is the latest instant the terminal accepts. It keeps time in
// a signed 32-bit counter, so anything from 2038-01-01 onwards is refused
// with "timeFormatError" — and, when it arrives via Modify, with the far less
// helpful "badParameters". Validity windows are clamped rather than rejected:
// a membership that runs past 2037 is not a reason to fail an enrollment.
var maxDeviceTime = time.Date(2037, time.December, 31, 23, 59, 59, 0, time.UTC)

func clampToDeviceRange(t time.Time) time.Time {
	if t.After(maxDeviceTime) {
		return maxDeviceTime
	}
	return t
}

// personPayload is the UserInfo body shared by Record (create) and Modify.
//
// doorRight and RightPlan are the part that is easy to miss: a person created
// without them is stored fine and their fingerprint enrolls fine, but every
// scan is denied at the door — the terminal reads the finger and then refuses
// because the person has no access schedule. planTemplateNo "1" is the
// device's built-in all-day template.
func personPayload(p Person) map[string]any {
	return map[string]any{
		"UserInfo": map[string]any{
			"employeeNo": p.EmployeeNo,
			"name":       p.Name,
			"userType":   "normal",
			"Valid": map[string]any{
				"enable":    true,
				"beginTime": clampToDeviceRange(p.ValidFrom).Format(deviceTimeLayout),
				"endTime":   clampToDeviceRange(p.ValidTo).Format(deviceTimeLayout),
				"timeType":  "local",
			},
			"doorRight": "1",
			"RightPlan": []map[string]any{
				{"doorNo": 1, "planTemplateNo": "1"},
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
	body, err := json.Marshal(map[string]any{
		"UserInfoSearchCond": map[string]any{
			"searchID":             "gym-go",
			"searchResultPosition": 0,
			"maxResults":           1,
			"EmployeeNoList":       []map[string]string{{"employeeNo": employeeNo}},
		},
	})
	if err != nil {
		return nil, err
	}

	respBody, status, err := c.do("POST", "/ISAPI/AccessControl/UserInfo/Search?format=json", "application/json", body)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("search person: device returned HTTP %s", httpStatusText(status))
	}

	var parsed struct {
		UserInfoSearch struct {
			NumOfMatches int `json:"numOfMatches"`
			UserInfo     []struct {
				EmployeeNo string `json:"employeeNo"`
				Name       string `json:"name"`
			} `json:"UserInfo"`
		} `json:"UserInfoSearch"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("search person: unexpected response: %s", string(respBody))
	}
	if len(parsed.UserInfoSearch.UserInfo) == 0 {
		return nil, nil
	}

	found := parsed.UserInfoSearch.UserInfo[0]
	return &Person{EmployeeNo: found.EmployeeNo, Name: found.Name}, nil
}

// jsonStatusCall performs a JSON call whose response is a bare
// statusCode/statusString ack.
func (c *Client) jsonStatusCall(method, path string, body []byte, op string) error {
	respBody, status, err := c.do(method, path, "application/json", body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("%s: device returned HTTP %s: %s", op, httpStatusText(status), string(respBody))
	}

	var parsed isapiStatus
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("%s: unexpected response: %s", op, string(respBody))
	}
	return parsed.err(op)
}
