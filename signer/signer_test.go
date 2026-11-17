package signer_test

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // MyPal's Signature Description v2 is SHA1WithRSA.
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/giangtv270304/gomypal/signer"
)

type vector struct {
	Name       string `json:"name"`
	AppID      string `json:"appId"`
	Nonce      string `json:"nonce"`
	Timestamp  string `json:"timestamp"`
	UID        string `json:"uid"`
	Version    string `json:"version"`
	Body       string `json:"body"`
	Canonical  string `json:"canonical"`
	PrivateKey string `json:"privateKey"`
	Signature  string `json:"signature"`
}

func loadVectors(t *testing.T) []vector {
	t.Helper()

	raw, err := os.ReadFile("signature-vectors.json")
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}

	var vectors []vector
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatalf("decode vectors: %v", err)
	}

	if len(vectors) == 0 {
		t.Fatal("no vectors")
	}

	return vectors
}

func (v vector) params(t *testing.T) signer.Params {
	t.Helper()

	millis, err := strconv.ParseInt(v.Timestamp, 10, 64)
	if err != nil {
		t.Fatalf("timestamp %q: %v", v.Timestamp, err)
	}

	var body []byte
	if v.Body != "" {
		body = []byte(v.Body)
	}

	return signer.Params{
		AppID:     v.AppID,
		Nonce:     v.Nonce,
		Timestamp: time.UnixMilli(millis),
		UID:       v.UID,
		Version:   v.Version,
		Body:      body,
	}
}

func TestSignVectors(t *testing.T) {
	t.Parallel()

	for _, testCase := range loadVectors(t) {
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()

			params := testCase.params(t)

			canonical, err := signer.Canonical(params)
			if err != nil {
				t.Fatalf("Canonical: %v", err)
			}

			if canonical != testCase.Canonical {
				t.Fatalf("canonical = %q, want %q", canonical, testCase.Canonical)
			}

			if testCase.Signature == "" {
				return
			}

			key, err := signer.ParsePrivateKey(testCase.PrivateKey)
			if err != nil {
				t.Fatalf("ParsePrivateKey: %v", err)
			}

			headers, err := signer.Headers(key, params)
			if err != nil {
				t.Fatalf("Headers: %v", err)
			}

			if got := headers[signer.HeaderSign]; got != testCase.Signature {
				t.Errorf("signature = %q, want %q", got, testCase.Signature)
			}

			want := map[string]string{
				signer.HeaderAppID:     testCase.AppID,
				signer.HeaderNonce:     testCase.Nonce,
				signer.HeaderTimestamp: testCase.Timestamp,
				signer.HeaderUID:       testCase.UID,
				signer.HeaderVersion:   testCase.Version,
			}
			for name, value := range want {
				if got, present := headers[name]; !present || got != value {
					t.Errorf("header %s = %q (present %v), want %q", name, got, present, value)
				}
			}

			signature, _ := base64.StdEncoding.DecodeString(headers[signer.HeaderSign])
			digest := sha1.Sum([]byte(testCase.Canonical)) //nolint:gosec // MyPal's Signature Description v2 is SHA1WithRSA.

			if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA1, digest[:], signature); err != nil {
				t.Errorf("signature does not verify with the public key: %v", err)
			}
		})
	}
}

func TestCanonical_WhitespaceAndKeyOrderDoNotMatter(t *testing.T) {
	t.Parallel()

	base := signer.Params{AppID: "app", Nonce: "N", Timestamp: time.UnixMilli(1), UID: "", Version: "v2", Body: nil}

	compact := base
	compact.Body = []byte(`{"b":1.50,"a":{"y":"2","x":true}}`)

	spaced := base
	spaced.Body = []byte("{ \"a\" : { \"x\" : true , \"y\" : \"2\" } ,\n \"b\" : 1.50 }")

	first, err := signer.Canonical(compact)
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}

	second, err := signer.Canonical(spaced)
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}

	want := "appId=app&nonce=N&timestamp=1&uid=&version=v2a={x:true,y:2}&b=1.50"
	if first != want || second != want {
		t.Errorf("canonical = %q and %q, want %q", first, second, want)
	}
}

func TestCanonical_RejectsWhatCannotBeSigned(t *testing.T) {
	t.Parallel()

	testCases := map[string]error{
		`[1,2]`:             signer.ErrBodyNotObject,
		`"text"`:            signer.ErrBodyNotObject,
		`{"a":`:             signer.ErrBodyNotObject,
		`null`:              signer.ErrBodyNotObject,
		`{"ids":[1,2]}`:     signer.ErrArrayNotSupported,
		`{"a":{"ids":[1]}}`: signer.ErrArrayNotSupported,
	}

	for body, want := range testCases {
		_, err := signer.Canonical(signer.Params{
			AppID: "app", Nonce: "N", Timestamp: time.UnixMilli(1), UID: "", Version: "v2", Body: []byte(body),
		})
		if !errors.Is(err, want) {
			t.Errorf("body %s: error = %v, want %v", body, err, want)
		}
	}
}

func TestParsePrivateKey_ErrorHasNoKeyMaterial(t *testing.T) {
	t.Parallel()

	vendor := loadVectors(t)[0].PrivateKey

	for _, bad := range []string{"", "@@@", vendor[:200], "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----"} {
		_, err := signer.ParsePrivateKey(bad)
		if !errors.Is(err, signer.ErrInvalidPrivateKey) {
			t.Errorf("error = %v, want ErrInvalidPrivateKey", err)

			continue
		}

		if len(bad) >= 8 && strings.Contains(err.Error(), bad[:8]) {
			t.Errorf("error %q contains key material", err)
		}
	}

	wrapped := vendor[:64] + "\n" + vendor[64:] + "\n"
	if _, err := signer.ParsePrivateKey(wrapped); err != nil {
		t.Errorf("key with line breaks: %v", err)
	}
}

func TestNewNonce(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}

	for range 100 {
		nonce, err := signer.NewNonce()
		if err != nil {
			t.Fatalf("NewNonce: %v", err)
		}

		if len(nonce) != 32 || strings.Trim(nonce, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") != "" {
			t.Fatalf("nonce = %q, want 32 characters A-Z0-9", nonce)
		}

		if seen[nonce] {
			t.Fatalf("nonce %q repeated", nonce)
		}

		seen[nonce] = true
	}
}
