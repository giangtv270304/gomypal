package signer_test

import (
	"testing"

	"github.com/gomypal/signer"
)

// The golden vector below comes verbatim from MyPal's "Signature
// Description v2" worked example. It was independently verified with
// `openssl dgst -sha1 -sign` against docPrivateKey before being written
// into this test — this is the single strongest correctness gate in the
// whole SDK, since it's checked against ground truth, not just internal
// consistency.
const (
	docAppID     = "3a9ea1263e624adf88518d4d88369aad"
	docNonce     = "TD4ENK2SN25HQBGPGTS5A1LSHCGZL4XR"
	docTimestamp = "1764252493756"
	docUID       = "102130"
	docVersion   = "v2"

	docSignString = "appId=3a9ea1263e624adf88518d4d88369aad&nonce=TD4ENK2SN25HQBGPGTS5A1LSHCGZL4XR&timestamp=1764252493756&uid=102130&version=v2email=zhangshuaishuai9527@gmail.com&productId=100004&transaction={balance:100,merchantName:mypal,transType:170}&userBankcardId=100168"

	docPrivateKey = "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCROf/gagyT3VVEhjelieoqvsEe4QW6KIE+hxXpIGCXYtO2w38rk5ht0VYWLX4dtmFf3vhkzIr2BzRe5cTetKbfPyfLSPhuyxnaSmJC0CVhvQpYsQWnDUQO1xp9N2BrOsW2sN5TTsmqklxZkVFiY4wn1QNo5RyrGQoE8coRvFO5tGcC7ygWuLFy/OQkZ31xOp6DdG3wBXYX/Ok+mjDBfgfZvNwV5mw5y7efoLbZQjlL9P5EYY8m6iSFw0nhZHT1a1gCZKk3lZlNhSPJ8z+KUOTAHV2qoQgygB0HvcIAuR5mFFp62jqRUE2HNRg9nWO5t/x84ShSMradNnRqgV8SiXr9AgMBAAECggEAPx8LdToCz34DIq6Hnz7LddrFS6uYdqpGSqwiPNqiwQnmZWihbIpRVWIIXclWKInt1uKt6fnxaDAHdju+DuJMKXpkzHmh6V4DlZqBDPM1XD7zMUt33nzomN6rl8Mq+FBqQRas+16iTq4gz5lW20HTWDjAq6XxiPhwH2gMYRsO9QaoilzTxpgeFqGBj1DUDDdGyJjdk+ibI3WXazeEEse20x+D7aK4IeGAMbCDHxkw3r1Bcm/3OqVrsMi9ex1V1YFfF1ElFeuHPEhUjCWMdUthWdHuZeIpMrAeNUyYbXzt60x0aDdSNFkj6DdRybhxxouJdMA9ccqU/5khzxpbhfrxTQKBgQDYT1aBWCfTD4cQt+4YqDBaqhBQBeR7Vb3xvqpkiS0rrp9qUExjHPiITI2r1GKnu0vLnw+V+eMcaJZcjVXy6DFjHC9wJEaGhnjkTKDvNw7jpnOikkE4/dT6m2K9MSUooNXtO0ViwenDQ2hjhuDuWf/JpOKqt58jBWUJ6PPAWctbiwKBgQCr368i06IASlqMTP0UZQA1WFH89ejM2gT91mKSCxOGqnFCKledBv1HuRcwFe/qpY4U8iXgT9Q5XvLnRbSuegzmBO7XNp99Qmjs5uk/hUXhhsYekSp/JjSMqoDyeer6s9B4nr/J+1qRHm3KzdAsAvgNdFaCZxaxRq4383bb28L0lwKBgQC5OMaEHy6KxkMLh9PzfSFaoUEypLkO+06YkIFrcSMiwnZzHKssO4R9gb/WIRkRM4olo+A+tilJDC2ktsmgJD8V9O6SAh3512/xXcR40haHzOI7lS0QVhpnNWMJSendrktAxn4VoKJAt5nNj5NPxfUHf48wS+eSn23zXGxMixddHQKBgAU6CzBq8S1DFZfYq4WYQLZelje0Hr1EEAzMHTlFvvVEb6dMdFehI8ivl0XuFNq06Co45u/4XYLVIVvpIcKAka3ZDo5a6eS8RUaXCxbUbbibCaFaVsmBFu4ELWGyVXQETd2tLjXtP+bT3xDgy2jbOkHmfM8AZY9Ej/3BmKa8+yPHAoGAdTn8MPdT4tOcFulIBgQ8Mljc3OAi4xkAZpa7dhgdVOJFIE/Gcj33ioe4loUO8M/EimWB1rdZUCZpxI+ZMwyWjzg5UU+uGktMVdDJ89EhHReDvyIo7mPez0JrhkumtsoUxUSGy/V0RQ9jXYvOPyCY/BUvgj7eaw1obhYU3/+rpvk="

	docExpectedSign = "Z9M/W1aUJhtI1O21DzGKTGW1c44EPWxgeDE1t0/FbHx+ddLrGe5809NEmaDJa//vj2n+P/mx01XTEre36BDGDbgIBkufudM2tMUrxXH3xtb83pVSXa1HFvpgHK73zhBqYxgPm5qLQCFDRJMngvucMAQbEctlNzjBzlAgKq6WcC42+/F3/9JfMYPvgzArBoSnLxMsucRyay5OT1D5iJmcGa0FwfE9g8IsCCsfMHbycc6BHTSsRlOCIrxIX3CCmIHxAH5RvV4xxuKOqG5+WafDtkKn9PvvfvugkrFVlIR/xxInVCae5W4LRisuuxECpQ7ReTqUx6dCwUzO/j39d65TkQ=="
)

func docBody() map[string]any {
	return map[string]any{
		"productId": float64(100004),
		"email":     "zhangshuaishuai9527@gmail.com",
		"userBankcardId": float64(100168),
		"transaction": map[string]any{
			"transType":    float64(170),
			"merchantName": "mypal",
			"balance":      "100",
		},
	}
}

func TestBuildSignString_GoldenVector(t *testing.T) {
	headerFields := map[string]string{
		"appId":     docAppID,
		"nonce":     docNonce,
		"timestamp": docTimestamp,
		"uid":       docUID,
		"version":   docVersion,
	}

	got := signer.BuildSignString(headerFields, docBody())
	if got != docSignString {
		t.Errorf("BuildSignString() =\n%q\nwant\n%q", got, docSignString)
	}
}

func TestSign_GoldenVector(t *testing.T) {
	key, err := signer.ParsePrivateKey(docPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}

	got, err := signer.Sign(key, docSignString)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if got != docExpectedSign {
		t.Errorf("Sign() = %q, want %q", got, docExpectedSign)
	}
}

func TestBuildSignString_HeaderIncludesEmptyUID(t *testing.T) {
	headerFields := map[string]string{
		"appId":     docAppID,
		"nonce":     docNonce,
		"timestamp": docTimestamp,
		"uid":       "",
		"version":   docVersion,
	}

	got := signer.BuildSignString(headerFields, nil)
	want := "appId=" + docAppID + "&nonce=" + docNonce + "&timestamp=" + docTimestamp + "&uid=&version=" + docVersion
	if got != want {
		t.Errorf("BuildSignString() = %q, want %q", got, want)
	}
}

func TestCanonicalBodyString_SkipsEmptyFields(t *testing.T) {
	headerFields := map[string]string{"appId": "a", "nonce": "n", "timestamp": "t", "uid": "", "version": "v"}

	body := map[string]any{
		"kept":    "value",
		"empty":   "",
		"nilVal":  nil,
		"zeroInt": float64(0), // zero-value numbers are NOT treated as empty
	}

	got := signer.BuildSignString(headerFields, body)
	want := "appId=a&nonce=n&timestamp=t&uid=&version=vkept=value&zeroInt=0"
	if got != want {
		t.Errorf("BuildSignString() = %q, want %q", got, want)
	}
}

func TestCanonicalBodyString_KeyOrderIndependent(t *testing.T) {
	headerFields := map[string]string{"appId": "a", "nonce": "n", "timestamp": "t", "uid": "", "version": "v"}

	bodyA := map[string]any{"z": "1", "a": "2", "m": "3"}
	bodyB := map[string]any{"m": "3", "z": "1", "a": "2"}

	gotA := signer.BuildSignString(headerFields, bodyA)
	gotB := signer.BuildSignString(headerFields, bodyB)
	if gotA != gotB {
		t.Errorf("BuildSignString() not order-independent: %q != %q", gotA, gotB)
	}
}

func TestParsePrivateKey_AcceptsBareBase64AndPEM(t *testing.T) {
	keyFromBase64, err := signer.ParsePrivateKey(docPrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey(bare base64): %v", err)
	}

	pemForm := "-----BEGIN PRIVATE KEY-----\n" + docPrivateKey + "\n-----END PRIVATE KEY-----\n"
	keyFromPEM, err := signer.ParsePrivateKey(pemForm)
	if err != nil {
		t.Fatalf("ParsePrivateKey(PEM): %v", err)
	}

	if keyFromBase64.D.Cmp(keyFromPEM.D) != 0 {
		t.Error("bare base64 and PEM forms parsed to different keys")
	}
}
