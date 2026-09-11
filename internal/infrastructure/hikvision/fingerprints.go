package hikvision

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrFingerprintBusy is the terminal's "Device Error" on capture. In practice
// it means no finger was on the sensor when the request arrived: the call is
// synchronous and the device does not wait for one.
var ErrFingerprintBusy = errors.New("the reader saw no finger within its wait window")

// DuplicateFingerprintError is returned when the terminal refuses to enroll a
// template because that same finger already belongs to somebody else. The
// device names the current owner, which is the whole reason this is a typed
// error: the caller can turn OwnerEmployeeNo into a member's name.
//
// This is also the answer to "what if two members enroll the same finger":
// they can't. The first enrollment wins and the second one lands here.
type DuplicateFingerprintError struct {
	OwnerEmployeeNo string
}

func (e *DuplicateFingerprintError) Error() string {
	return fmt.Sprintf("fingerprint already enrolled for employee %s", e.OwnerEmployeeNo)
}

// CapturedFingerprint is one live scan taken off the terminal's sensor.
type CapturedFingerprint struct {
	// Template is the base64 template, normalized (see normalizeBase64).
	Template string
	FingerNo int
	Quality  int
}

// captureFingerPrintResponse mirrors the XML the device answers with.
type captureFingerPrintResponse struct {
	XMLName    xml.Name `xml:"CaptureFingerPrint"`
	FingerData string   `xml:"fingerData"`
	FingerNo   int      `xml:"fingerNo"`
	Quality    int      `xml:"fingerPrintQuality"`
}

// deviceFingerWait is how long the terminal itself waits for a finger before
// answering "Device Error", measured against a DS-K1T8003MF: about eleven
// seconds. It cannot be configured, which is why covering a longer window
// means chaining attempts rather than asking the device to wait.
const deviceFingerWait = 11 * time.Second

// CaptureFingerprintWithin keeps asking the terminal to read a finger until
// somebody presents one or the window runs out.
//
// The window exists because of where these readers get installed. At the
// first gym the terminal sits three metres from the desk, out of sight of
// the screen, so the member has to walk over after the receptionist starts
// the capture — and the device gives up on its own after ~11 seconds.
// Chaining attempts turns that into as long as the caller is willing to
// wait. Each attempt is one request per 11 seconds, gentler than the event
// poller, so this does not stress the device.
//
// Note the reader is busy enrolling for the whole window: keep it to the
// time a person actually needs to walk over, not minutes.
func (c *Client) CaptureFingerprintWithin(fingerNo int, window time.Duration) (*CapturedFingerprint, error) {
	deadline := time.Now().Add(window)

	for {
		captured, err := c.CaptureFingerprint(fingerNo)
		if err == nil {
			return captured, nil
		}
		// Only "nobody put a finger down" is worth waiting through. A real
		// fault should surface at once instead of being retried for a minute.
		if !errors.Is(err, ErrFingerprintBusy) {
			return nil, err
		}
		if !time.Now().Before(deadline) {
			return nil, err
		}

		// A healthy device paces this loop by itself, since each attempt
		// blocks for its own wait. The pause is for the case where it stops
		// doing that — a sensor fault answering instantly would otherwise
		// turn this into thousands of requests over the window.
		time.Sleep(captureRetryPause)
	}
}

// captureRetryPause is negligible next to the device's own wait and bounds
// the loop if that wait ever collapses.
const captureRetryPause = 500 * time.Millisecond

// CaptureFingerprint asks the terminal to read a finger, once.
//
// Three quirks, all found the hard way against a DS-K1T8003MF (V1.4.1):
//
//   - This endpoint speaks XML only. Sending the same request as JSON with
//     ?format=json is rejected with "MessageParametersLack", even though the
//     neighbouring person and fingerprint endpoints are JSON.
//   - The device blocks for about eleven seconds waiting for a finger and
//     then answers "Device Error". An HTTP timeout shorter than that never
//     sees the answer and reports a connection failure instead — which is
//     how "no finger was presented" reached the gym disguised as "error de
//     comunicación con el huellero".
//   - Eleven seconds is the device's own limit; use CaptureFingerprintWithin
//     when the person needs longer to reach the reader.
func (c *Client) CaptureFingerprint(fingerNo int) (*CapturedFingerprint, error) {
	body := []byte(fmt.Sprintf("<CaptureFingerPrintCond><fingerNo>%d</fingerNo></CaptureFingerPrintCond>", fingerNo))

	respBody, status, err := c.doUsing(c.waitClient, "POST", "/ISAPI/AccessControl/CaptureFingerPrint", "application/xml", body)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("capture fingerprint: device returned HTTP %s", httpStatusText(status))
	}

	var parsed captureFingerPrintResponse
	if err := xml.Unmarshal(respBody, &parsed); err != nil || parsed.FingerData == "" {
		// A failed capture answers with <ResponseStatus> instead, which does
		// not unmarshal into the struct above.
		if strings.Contains(string(respBody), "deviceError") {
			return nil, ErrFingerprintBusy
		}
		return nil, fmt.Errorf("capture fingerprint: unexpected response: %s", string(respBody))
	}

	template, err := normalizeBase64(parsed.FingerData)
	if err != nil {
		return nil, fmt.Errorf("capture fingerprint: %w", err)
	}

	return &CapturedFingerprint{
		Template: template,
		FingerNo: parsed.FingerNo,
		Quality:  parsed.Quality,
	}, nil
}

// normalizeBase64 repairs the padding of a base64 string.
//
// The terminal sometimes emits a template whose base64 length is not a
// multiple of 4 (a 685-character string was observed for a 513-byte
// template). Its own enrollment endpoint then rejects the value it just
// produced. Re-padding fixes it; do NOT "fix" it by trimming data bytes —
// that stores a template that enrolls without complaint and then never
// matches a finger.
func normalizeBase64(s string) (string, error) {
	s = strings.TrimSpace(s)
	if rem := len(s) % 4; rem != 0 {
		s += strings.Repeat("=", 4-rem)
	}
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		return "", fmt.Errorf("template is not valid base64: %w", err)
	}
	return s, nil
}

// fingerPrintStatusResponse is the enrollment ack.
type fingerPrintStatusResponse struct {
	FingerPrintStatus struct {
		Status     string `json:"status"`
		StatusList []struct {
			ID                   int    `json:"id"`
			CardReaderRecvStatus int    `json:"cardReaderRecvStatus"`
			ErrorMsg             string `json:"errorMsg"`
		} `json:"StatusList"`
	} `json:"FingerPrintStatus"`
}

// cardReaderRecvStatus values, as observed on a DS-K1T8003MF. Hikvision does
// not publish this enum outside its partner portal; these three were pinned
// down by enrolling, re-enrolling and cross-enrolling a real finger.
const (
	recvStatusAdded       = 0 // stored in a slot that was empty
	recvStatusOverwritten = 1 // replaced what that slot held
	recvStatusDuplicate   = 5 // this finger already belongs to errorMsg's employeeNo
)

// EnrollFingerprint stores a captured template against a person, in one of
// the terminal's ten finger slots (1-10).
//
// enableCardReader is the list of reader modules to write to and must be
// [1] on this single-reader terminal; 0 is rejected as an illegal reader
// number. fingerType "normalFP" is an ordinary finger, as opposed to the
// duress/patrol variants the device also supports.
func (c *Client) EnrollFingerprint(employeeNo string, fingerPrintID int, template string) error {
	template, err := normalizeBase64(template)
	if err != nil {
		return err
	}

	body, err := json.Marshal(map[string]any{
		"FingerPrintCfg": map[string]any{
			"employeeNo":       employeeNo,
			"enableCardReader": []int{1},
			"fingerPrintID":    fingerPrintID,
			"fingerType":       "normalFP",
			"fingerData":       template,
		},
	})
	if err != nil {
		return err
	}

	respBody, status, err := c.do("POST", "/ISAPI/AccessControl/FingerPrint/SetUp?format=json", "application/json", body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("enroll fingerprint: device returned HTTP %s: %s", httpStatusText(status), string(respBody))
	}

	var parsed fingerPrintStatusResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("enroll fingerprint: unexpected response: %s", string(respBody))
	}
	if len(parsed.FingerPrintStatus.StatusList) == 0 {
		return fmt.Errorf("enroll fingerprint: device returned no status: %s", string(respBody))
	}

	result := parsed.FingerPrintStatus.StatusList[0]
	switch result.CardReaderRecvStatus {
	case recvStatusAdded, recvStatusOverwritten:
		return nil
	case recvStatusDuplicate:
		return &DuplicateFingerprintError{OwnerEmployeeNo: result.ErrorMsg}
	default:
		return fmt.Errorf("enroll fingerprint: device reported status %d (%s)", result.CardReaderRecvStatus, result.ErrorMsg)
	}
}

// DeleteFingerprints removes every fingerprint belonging to a person and
// waits for the device to actually apply it.
//
// Deleting is asynchronous and the acknowledgement lies: the request returns
// {"status":"success"} while the templates are still in the reader, so an
// immediate re-enrollment of the same finger comes back as a duplicate. The
// real outcome only shows up in FingerPrint/DeleteProcess, which is what this
// polls.
//
// Note the shape: mode byCardReader with CardReaderDetail as a single object.
// The byEmployeeNo mode advertised in /capabilities is rejected as
// badParameters on this firmware whatever shape it is given.
func (c *Client) DeleteFingerprints(employeeNo string) error {
	body, err := json.Marshal(map[string]any{
		"FingerPrintDelete": map[string]any{
			"mode": "byCardReader",
			"CardReaderDetail": map[string]any{
				"cardReaderNo": 1,
				"clearAllCard": false,
				"employeeNo":   employeeNo,
			},
		},
	})
	if err != nil {
		return err
	}

	if err := c.jsonStatusCall("PUT", "/ISAPI/AccessControl/FingerPrint/Delete?format=json", body, "delete fingerprints"); err != nil {
		return err
	}
	return c.waitForFingerprintDelete()
}

// waitForFingerprintDelete polls DeleteProcess until the device stops
// reporting the delete as in progress.
func (c *Client) waitForFingerprintDelete() error {
	const (
		attempts = 10
		interval = 500 * time.Millisecond
	)

	var last string
	for i := 0; i < attempts; i++ {
		respBody, status, err := c.do("GET", "/ISAPI/AccessControl/FingerPrint/DeleteProcess?format=json", "", nil)
		if err != nil {
			return err
		}
		if status < 200 || status >= 300 {
			return fmt.Errorf("delete progress: device returned HTTP %s", httpStatusText(status))
		}

		var parsed struct {
			FingerPrintDeleteProcess struct {
				Status string `json:"status"`
			} `json:"FingerPrintDeleteProcess"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return fmt.Errorf("delete progress: unexpected response: %s", string(respBody))
		}

		last = parsed.FingerPrintDeleteProcess.Status
		switch last {
		case "success":
			return nil
		case "processing":
			time.Sleep(interval)
			continue
		default: // "failed", or anything unforeseen
			return fmt.Errorf("delete fingerprints: device reported %q", last)
		}
	}
	return fmt.Errorf("delete fingerprints: still %q after %v", last, time.Duration(attempts)*interval)
}
