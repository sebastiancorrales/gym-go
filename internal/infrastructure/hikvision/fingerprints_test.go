package hikvision

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

// TestNormalizeBase64PreservesEveryByte is the regression guard for the bug
// that cost the most time to find: a template whose base64 length is not a
// multiple of 4 must be re-padded, never trimmed. Dropping a data byte to
// make it "fit" produces a template the terminal accepts and stores without
// any error, and which then never matches the finger it came from.
func TestNormalizeBase64PreservesEveryByte(t *testing.T) {
	raw := []byte(strings.Repeat("\x01\x02\x03", 171)) // 513 bytes, like a real template
	encoded := base64.StdEncoding.EncodeToString(raw)

	// The device emits this with a stray character that breaks the padding.
	malformed := strings.TrimSuffix(encoded, "=")

	fixed, err := normalizeBase64(malformed)
	if err != nil {
		t.Fatalf("normalizeBase64: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(fixed)
	if err != nil {
		t.Fatalf("result is not valid base64: %v", err)
	}
	if len(decoded) != len(raw) {
		t.Fatalf("template lost data: got %d bytes, want %d", len(decoded), len(raw))
	}
	if !bytes.Equal(decoded, raw) {
		t.Error("template bytes changed during normalization")
	}
}

func TestNormalizeBase64RejectsGarbage(t *testing.T) {
	if _, err := normalizeBase64("not base64 at all!!"); err == nil {
		t.Error("expected an error for a non-base64 template")
	}
}

func TestNormalizeBase64LeavesValidInputAlone(t *testing.T) {
	valid := base64.StdEncoding.EncodeToString([]byte("hello world"))
	got, err := normalizeBase64(valid)
	if err != nil {
		t.Fatalf("normalizeBase64: %v", err)
	}
	if got != valid {
		t.Errorf("got %q, want unchanged %q", got, valid)
	}
}
