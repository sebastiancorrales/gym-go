package hikvision

import (
	"encoding/xml"
	"fmt"
)

// remoteControlDoorResponse mirrors ISAPI's <ResponseStatus> body, e.g.:
//
//	<ResponseStatus version="1.0" ...>
//	  <requestURL></requestURL>
//	  <statusCode>1</statusCode>
//	  <statusString>OK</statusString>
//	  <subStatusCode>ok</subStatusCode>
//	</ResponseStatus>
//
// statusCode 1 is the device's success code; anything else is a failure
// described by statusString/subStatusCode.
type remoteControlDoorResponse struct {
	XMLName       xml.Name `xml:"ResponseStatus"`
	StatusCode    int      `xml:"statusCode"`
	StatusString  string   `xml:"statusString"`
	SubStatusCode string   `xml:"subStatusCode"`
}

// OpenDoor sends the remote-open command for the given door number
// (DS-K1T8003MF and other single-door terminals use door 1). It releases
// the relay driving the electromagnetic lock exactly like a granted
// card/fingerprint match would.
func (c *Client) OpenDoor(doorNo int) error {
	path := fmt.Sprintf("/ISAPI/AccessControl/RemoteControl/door/%d", doorNo)
	body := []byte(`<RemoteControlDoor><cmd>open</cmd></RemoteControlDoor>`)

	respBody, status, err := c.do("PUT", path, "application/xml", body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("device returned HTTP %s: %s", httpStatusText(status), string(respBody))
	}

	var parsed remoteControlDoorResponse
	if err := xml.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("unexpected response from device: %s", string(respBody))
	}
	if parsed.StatusCode != 1 {
		return fmt.Errorf("device rejected the open command: %s (%s)", parsed.StatusString, parsed.SubStatusCode)
	}
	return nil
}
