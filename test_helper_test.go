package gomypal

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // matches MyPal's SHA1WithRSA spec
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gomypal/signer"
)

const testAppID = "test-app-id"

// newTestKey generates a throwaway RSA key pair and returns both the
// PKCS8-bare-base64 form NewClient expects and the *rsa.PrivateKey for
// verifySignature to check against.
func newTestKey(t *testing.T) (string, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	return base64.StdEncoding.EncodeToString(der), key
}

// newTestClient generates a fresh RSA key pair, builds the server
// handler via buildHandler (which receives the key's public half so it
// can call verifySignature), spins up an httptest.Server for it, and
// returns a *Client wired to that server. The public key must be handed
// to the handler builder rather than returned alongside *Client, because
// a "client, pub := newTestClient(t, func(...){ ...pub... })"-shaped call
// wouldn't compile: per the spec, pub's scope only begins at the end of
// that whole short variable declaration, so it isn't visible yet inside
// a closure literal that's part of the same statement's RHS.
func newTestClient(t *testing.T, buildHandler func(pub *rsa.PublicKey) http.HandlerFunc) *Client {
	t.Helper()

	keyRaw, key := newTestKey(t)
	handler := buildHandler(&key.PublicKey)

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client, err := NewClient(srv.URL, testAppID, keyRaw)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

// writeJSON writes v as a JSON response body with the given status code.
func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

// readBody reads and returns r's body, restoring it so the handler's
// caller (httptest's request plumbing) can still be inspected afterward.
func readBody(t *testing.T, r *http.Request) []byte {
	t.Helper()

	if r.Body == nil {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

// verifySignature recomputes the expected signString from r's actual
// headers + body and verifies r's sign header against pub using
// RSA-SHA1 PKCS1v15 — proving the whole Client -> signer pipeline
// produces a signature the server could actually verify.
func verifySignature(t *testing.T, r *http.Request, rawBody []byte, pub *rsa.PublicKey) {
	t.Helper()

	headerFields := map[string]string{
		"appId":     r.Header.Get("appId"),
		"nonce":     r.Header.Get("nonce"),
		"timestamp": r.Header.Get("timestamp"),
		"uid":       r.Header.Get("uid"), // "" when omitted, matches signing rule
		"version":   r.Header.Get("version"),
	}
	if headerFields["appId"] == "" || headerFields["nonce"] == "" || headerFields["timestamp"] == "" {
		t.Fatal("missing required signing headers")
	}

	var body map[string]any
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
	}

	signString := signer.BuildSignString(headerFields, body)
	digest := sha1.Sum([]byte(signString)) //nolint:gosec // matches MyPal's SHA1WithRSA spec

	sigBytes, err := base64.StdEncoding.DecodeString(r.Header.Get("sign"))
	if err != nil {
		t.Fatalf("decode sign header: %v", err)
	}

	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA1, digest[:], sigBytes); err != nil {
		t.Errorf("signature verification failed: %v (signString=%q)", err, signString)
	}
}

// recordingHTTPClient is a fake HTTPClient that captures the last
// request it was asked to send (and its body, since http.Request.Body
// can only be read once) and returns a canned response. Used by unit
// tests that need to inspect exactly what Client would send over the
// wire without spinning up a real server.
type recordingHTTPClient struct {
	req     *http.Request
	reqBody []byte

	status int
	body   []byte
	err    error
}

func (c *recordingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.req = req
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		c.reqBody = b
	}

	if c.err != nil {
		return nil, c.err
	}

	status := c.status
	if status == 0 {
		status = http.StatusOK
	}

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(c.body)),
		Header:     make(http.Header),
	}, nil
}
