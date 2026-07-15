package gomypal

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// ── NewClient validation ────────────────────────────────────────────────────

func TestNewClient_Validation(t *testing.T) {
	validKey := newTestKeyForClientTest(t)

	tests := []struct {
		name       string
		baseURL    string
		appID      string
		privateKey string
		wantErr    error
	}{
		{"empty base URL", "", "app", validKey, ErrEmptyBaseURL},
		{"empty app id", "https://sandbox.mypal.example", "", validKey, ErrEmptyAppID},
		{"empty private key", "https://sandbox.mypal.example", "app", "", ErrEmptyPrivateKey},
		{"invalid private key", "https://sandbox.mypal.example", "app", "not-a-valid-key", ErrInvalidPrivateKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.baseURL, tt.appID, tt.privateKey)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewClient() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewClient_Defaults(t *testing.T) {
	key := newTestKeyForClientTest(t)
	c, err := NewClient("https://sandbox.mypal.example", "app", key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.httpClient != http.DefaultClient {
		t.Error("expected default httpClient to be http.DefaultClient")
	}
	if c.debug {
		t.Error("expected debug to default to false")
	}
	if c.version != "v2" {
		t.Errorf("version = %q, want %q", c.version, "v2")
	}
}

func TestNewClient_OptionsApplied(t *testing.T) {
	key := newTestKeyForClientTest(t)
	rc := &recordingHTTPClient{}
	c, err := NewClient("https://sandbox.mypal.example", "app", key,
		WithHTTPClient(rc),
		WithDebug(true),
		WithVersion("v3"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.httpClient != rc {
		t.Error("WithHTTPClient option was not applied")
	}
	if !c.debug {
		t.Error("WithDebug(true) option was not applied")
	}
	if c.version != "v3" {
		t.Errorf("version = %q, want %q", c.version, "v3")
	}
}

// ── buildRequest / excute ───────────────────────────────────────────────────

func TestBuildRequest_NilRequest(t *testing.T) {
	key := newTestKeyForClientTest(t)
	c, err := NewClient("https://sandbox.mypal.example", "app", key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := c.buildRequest(nil); !errors.Is(err, ErrNilRequest) {
		t.Errorf("buildRequest(nil) error = %v, want %v", err, ErrNilRequest)
	}
}

func TestBuildRequest_SetsContentTypeAndUserAgent(t *testing.T) {
	key := newTestKeyForClientTest(t)
	rc := &recordingHTTPClient{status: http.StatusOK, body: []byte(`{}`)}
	c, err := NewClient("https://sandbox.mypal.example", "app", key, WithHTTPClient(rc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := &request{Method: http.MethodPost, Endpoint: "/user/register", Params: map[string]any{"email": "a@b.com"}}
	if _, err := c.excute(context.Background(), req); err != nil {
		t.Fatalf("excute: %v", err)
	}

	if got := rc.req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := rc.req.Header.Get("User-Agent"); got != UserAgent {
		t.Errorf("User-Agent = %q, want %q", got, UserAgent)
	}
}

func TestBuildRequest_NilParamsSendsEmptyObjectBody(t *testing.T) {
	key := newTestKeyForClientTest(t)
	rc := &recordingHTTPClient{status: http.StatusOK, body: []byte(`{}`)}
	c, err := NewClient("https://sandbox.mypal.example", "app", key, WithHTTPClient(rc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := &request{Method: http.MethodPost, Endpoint: "/user/kyc/status", UID: "1"}
	if _, err := c.excute(context.Background(), req); err != nil {
		t.Fatalf("excute: %v", err)
	}

	if string(rc.reqBody) != "{}" {
		t.Errorf("request body = %q, want %q", rc.reqBody, "{}")
	}
}

func TestExcute_NonSuccessStatusReturnsError(t *testing.T) {
	key := newTestKeyForClientTest(t)
	rc := &recordingHTTPClient{status: http.StatusBadRequest, body: []byte(`{"message":"bad request"}`)}
	c, err := NewClient("https://sandbox.mypal.example", "app", key, WithHTTPClient(rc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := &request{Method: http.MethodPost, Endpoint: "/user/register", Params: map[string]any{"email": "a@b.com"}}
	_, err = c.excute(context.Background(), req)
	if !errors.Is(err, ErrUnexpectedStatus) {
		t.Errorf("excute() error = %v, want wrapping %v", err, ErrUnexpectedStatus)
	}
}

func TestExcute_TransportError(t *testing.T) {
	key := newTestKeyForClientTest(t)
	rc := &recordingHTTPClient{err: errors.New("connection refused")}
	c, err := NewClient("https://sandbox.mypal.example", "app", key, WithHTTPClient(rc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := &request{Method: http.MethodPost, Endpoint: "/user/register", Params: map[string]any{"email": "a@b.com"}}
	_, err = c.excute(context.Background(), req)
	if !errors.Is(err, ErrHTTPFailure) {
		t.Errorf("excute() error = %v, want wrapping %v", err, ErrHTTPFailure)
	}
}

// newTestKeyForClientTest wraps newTestKey to only keep the string form,
// since most tests here just need a syntactically valid key to get past
// NewClient's validation.
func newTestKeyForClientTest(t *testing.T) string {
	t.Helper()
	keyRaw, _ := newTestKey(t)
	return keyRaw
}
