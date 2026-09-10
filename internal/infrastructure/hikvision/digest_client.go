// Package hikvision talks to Hikvision access-control terminals (e.g. the
// DS-K1T8003MF) over ISAPI — the plain-HTTP API Hikvision ships on its
// network devices, authenticated with RFC 2617 Digest auth. Unlike the
// DigitalPersona fingerprint reader (see internal/usecases/biometric_usecase.go),
// these terminals are network devices, not USB peripherals, so no C#
// helper process or CGO is needed: Go's net/http talks to them directly.
package hikvision

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client is a minimal ISAPI HTTP client with Digest authentication. It is
// intentionally narrow — one device, one credential pair — matching how a
// Device row's HikvisionConfig is scoped.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewClient builds a client for the terminal at ip:port.
func NewClient(ip string, port int, username, password string) *Client {
	return &Client{
		baseURL:    fmt.Sprintf("http://%s:%d", ip, port),
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// do performs method+path with body, retrying once with a Digest
// Authorization header computed from the challenge in the first response's
// WWW-Authenticate header (the standard RFC 2617 handshake — Hikvision
// devices reject ISAPI requests sent without it with a 401).
func (c *Client) do(method, path, contentType string, body []byte) ([]byte, int, error) {
	url := c.baseURL + path

	req, err := newRequest(method, url, contentType, body)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot reach device: %w", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		return respBody, resp.StatusCode, err
	}

	challenge := resp.Header.Get("WWW-Authenticate")
	resp.Body.Close()
	if challenge == "" {
		return nil, resp.StatusCode, fmt.Errorf("device returned 401 without a WWW-Authenticate challenge")
	}

	authHeader, err := c.buildDigestHeader(method, path, challenge)
	if err != nil {
		return nil, 0, fmt.Errorf("building digest auth: %w", err)
	}

	req2, err := newRequest(method, url, contentType, body)
	if err != nil {
		return nil, 0, err
	}
	req2.Header.Set("Authorization", authHeader)

	resp2, err := c.httpClient.Do(req2)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot reach device: %w", err)
	}
	defer resp2.Body.Close()

	respBody, err := io.ReadAll(resp2.Body)
	return respBody, resp2.StatusCode, err
}

func newRequest(method, url, contentType string, body []byte) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = strings.NewReader(string(body))
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

// buildDigestHeader implements the qop=auth case of RFC 2617, which is what
// Hikvision's embedded HTTP server sends. It deliberately does not handle
// qop=auth-int (request-body hashing) — ISAPI doesn't ask for it.
func (c *Client) buildDigestHeader(method, uri, challenge string) (string, error) {
	params := parseDigestChallenge(challenge)

	realm := params["realm"]
	nonce := params["nonce"]
	opaque := params["opaque"]
	qop := params["qop"]
	if realm == "" || nonce == "" {
		return "", fmt.Errorf("incomplete Digest challenge: %q", challenge)
	}

	cnonce, err := randomHex(16)
	if err != nil {
		return "", err
	}
	nc := "00000001"

	ha1 := md5Hex(fmt.Sprintf("%s:%s:%s", c.username, realm, c.password))
	ha2 := md5Hex(fmt.Sprintf("%s:%s", method, uri))

	var response string
	if qop != "" {
		// qop is a quoted comma-list (e.g. "auth"); Hikvision only offers "auth".
		response = md5Hex(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, "auth", ha2))
	} else {
		response = md5Hex(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
	}

	header := fmt.Sprintf(
		`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
		c.username, realm, nonce, uri, response,
	)
	if qop != "" {
		header += fmt.Sprintf(`, qop=auth, nc=%s, cnonce="%s"`, nc, cnonce)
	}
	if opaque != "" {
		header += fmt.Sprintf(`, opaque="%s"`, opaque)
	}
	return header, nil
}

// parseDigestChallenge parses a `WWW-Authenticate: Digest key="value", ...`
// header into its key/value pairs (unquoted).
func parseDigestChallenge(header string) map[string]string {
	header = strings.TrimPrefix(strings.TrimSpace(header), "Digest ")
	params := map[string]string{}
	for _, part := range splitDigestParams(header) {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
		params[key] = val
	}
	return params
}

// splitDigestParams splits on commas that are not inside a quoted value —
// a plain strings.Split would break on the comma inside qop="auth,auth-int".
func splitDigestParams(s string) []string {
	var parts []string
	inQuotes := false
	start := 0
	for i, r := range s {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// httpStatusText is a small helper used by callers that want to surface the
// raw status code in an error message.
func httpStatusText(code int) string {
	return strconv.Itoa(code) + " " + http.StatusText(code)
}
