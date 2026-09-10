package hikvision

import "testing"

// TestDigestMath checks the MD5 pieces of the RFC 2617 qop=auth algorithm
// against the worked example from the RFC itself. This is the part that
// must be exactly right — a single stray colon or wrong field order here
// makes the device reject every request with 401, and that failure mode
// gives no useful diagnostic beyond "unauthorized".
func TestDigestMath(t *testing.T) {
	const (
		username = "Mufasa"
		password = "Circle Of Life"
		realm    = "testrealm@host.com"
		nonce    = "dcd98b7102dd2f0e8b11d0f600bfb0c093"
		nc       = "00000001"
		cnonce   = "0a4f113b"
		method   = "GET"
		uri      = "/dir/index.html"
	)

	ha1 := md5Hex(username + ":" + realm + ":" + password)
	if want := "939e7578ed9e3c518a452acee763bce9"; ha1 != want {
		t.Fatalf("HA1 = %s, want %s", ha1, want)
	}

	ha2 := md5Hex(method + ":" + uri)
	if want := "39aff3a2bab6126f332b942af96d3366"; ha2 != want {
		t.Fatalf("HA2 = %s, want %s", ha2, want)
	}

	response := md5Hex(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":auth:" + ha2)
	if want := "6629fae49393a05397450978507c4ef1"; response != want {
		t.Fatalf("response = %s, want %s", response, want)
	}
}

func TestParseDigestChallenge(t *testing.T) {
	header := `Digest qop="auth", realm="DS-K1T8003MF", nonce="abc123", opaque="def456"`
	params := parseDigestChallenge(header)

	for k, want := range map[string]string{
		"qop": "auth", "realm": "DS-K1T8003MF", "nonce": "abc123", "opaque": "def456",
	} {
		if got := params[k]; got != want {
			t.Errorf("params[%q] = %q, want %q", k, got, want)
		}
	}
}
