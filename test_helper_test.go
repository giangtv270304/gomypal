package gomypal_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // MyPal's Signature Description v2 is SHA1WithRSA.
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/giangtv270304/gomypal"
)

const (
	envBaseURL        = "MYPAL_BASE_URL"
	envAppID          = "MYPAL_APP_ID"
	envPrivateKey     = "MYPAL_PRIVATE_KEY"
	envPrivateKeyFile = "MYPAL_PRIVATE_KEY_FILE"
	envTestUID        = "MYPAL_TEST_UID"
	envTestCardID     = "MYPAL_TEST_CARD_ID"
	envTestProductID  = "MYPAL_TEST_PRODUCT_ID"

	testAppID     = "test-app-id"
	testUID       = "10001"
	testNonce     = "TESTNONCE0123456789ABCDEFGHIJKLM"
	testTimestamp = "1764252493756"

	uploadFileName = "front.png"

	successEmpty = `{"code":0,"msg":"success","data":{}}`
)

var (
	testKeyOnce   sync.Once
	testKey       *rsa.PrivateKey
	testKeyBase64 string
)

func throwawayKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	testKeyOnce.Do(func() {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}

		der, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			panic(err)
		}

		testKey = key
		testKeyBase64 = base64.StdEncoding.EncodeToString(der)
	})

	return testKey, testKeyBase64
}

func fixedClock() time.Time { return time.UnixMilli(1764252493756) }

func fixedNonce() (string, error) { return testNonce, nil }

type fakeAPI struct {
	path      string
	uid       string
	body      string
	multipart bool
	status    int
	response  string
}

func newFakeClient(t *testing.T, want fakeAPI, opts ...gomypal.Option) *gomypal.Client {
	t.Helper()

	key, keyBase64 := throwawayKey(t)

	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)

		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}

		checkFakeRequest(t, &key.PublicKey, want, r, raw)

		status := want.status
		if status == 0 {
			status = http.StatusOK
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, want.response)
	}))

	t.Cleanup(func() {
		server.Close()

		if got := calls.Load(); got != 1 {
			t.Errorf("fake server got %d requests, want 1", got)
		}
	})

	options := append([]gomypal.Option{gomypal.WithClock(fixedClock), gomypal.WithNonceSource(fixedNonce)}, opts...)

	client, err := gomypal.NewClient(server.URL+"/openapi", testAppID, keyBase64, options...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return client
}

func checkFakeRequest(t *testing.T, publicKey *rsa.PublicKey, want fakeAPI, r *http.Request, raw []byte) {
	t.Helper()

	if r.Method != http.MethodPost || r.URL.Path != "/openapi"+want.path || r.URL.RawQuery != "" {
		t.Errorf("request = %s %s?%s, want POST /openapi%s without query", r.Method, r.URL.Path, r.URL.RawQuery, want.path)
	}

	wantHeaders := map[string]string{
		"appId":      testAppID,
		"nonce":      testNonce,
		"timestamp":  testTimestamp,
		"uid":        want.uid,
		"version":    "v2",
		"User-Agent": "gomypal/" + gomypal.Version,
	}
	for name, value := range wantHeaders {
		if got := r.Header.Get(name); got != value {
			t.Errorf("header %s = %q, want %q", name, got, value)
		}
	}

	if _, present := r.Header["Uid"]; !present {
		t.Error("header uid is absent, want it present (empty when the endpoint has no uid)")
	}

	signedBody := raw

	if want.multipart {
		signedBody = nil

		checkFakeUpload(t, want, r, raw)
	} else {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		if string(raw) != want.body {
			t.Errorf("body = %s, want %s", raw, want.body)
		}
	}

	canonical := "appId=" + r.Header.Get("appId") + "&nonce=" + r.Header.Get("nonce") + "&timestamp=" + r.Header.Get("timestamp") +
		"&uid=" + r.Header.Get("uid") + "&version=" + r.Header.Get("version") + referenceBodyPart(t, signedBody)
	canonical = strings.ReplaceAll(canonical, `"`, "")

	signature, err := base64.StdEncoding.DecodeString(r.Header.Get("sign"))
	if err != nil {
		t.Errorf("sign header is not base64: %v", err)
	}

	digest := sha1.Sum([]byte(canonical)) //nolint:gosec // MyPal's Signature Description v2 is SHA1WithRSA.
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA1, digest[:], signature); err != nil {
		t.Errorf("signature does not verify over %q: %v", canonical, err)
	}
}

func checkFakeUpload(t *testing.T, want fakeAPI, r *http.Request, raw []byte) {
	t.Helper()

	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		t.Errorf("Content-Type = %q, want multipart/form-data", r.Header.Get("Content-Type"))

		return
	}

	part, err := multipart.NewReader(bytes.NewReader(raw), params["boundary"]).NextPart()
	if err != nil {
		t.Errorf("read multipart part: %v", err)

		return
	}

	content, _ := io.ReadAll(part)

	if part.FormName() != "idCard" || part.FileName() != uploadFileName || string(content) != want.body {
		t.Errorf("part = field %q file %q content %q, want idCard %s %q",
			part.FormName(), part.FileName(), content, uploadFileName, want.body)
	}
}

// referenceBodyPart is the test's own reading of spec section 2.2. It does not call the SDK signer.
func referenceBodyPart(t *testing.T, raw []byte) string {
	t.Helper()

	if len(raw) == 0 {
		return ""
	}

	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()

	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		t.Errorf("request body is not a JSON object: %v", err)

		return ""
	}

	delete(body, "sign")

	return referenceJoin(body, "=", "&")
}

func referenceJoin(fields map[string]any, keyValueSeparator, pairSeparator string) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))

	for _, key := range keys {
		switch value := fields[key].(type) {
		case nil:
		case string:
			if value != "" {
				pairs = append(pairs, key+keyValueSeparator+value)
			}
		case json.Number:
			pairs = append(pairs, key+keyValueSeparator+value.String())
		case bool:
			if value {
				pairs = append(pairs, key+keyValueSeparator+"true")
			} else {
				pairs = append(pairs, key+keyValueSeparator+"false")
			}
		case map[string]any:
			pairs = append(pairs, key+keyValueSeparator+"{"+referenceJoin(value, ":", ",")+"}")
		}
	}

	return strings.Join(pairs, pairSeparator)
}

func apiErrorOf(t *testing.T, err error) *gomypal.APIError {
	t.Helper()

	var apiErr *gomypal.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *gomypal.APIError", err)
	}

	return apiErr
}

func assertAPIError(t *testing.T, err error, wantOp string, wantStatus int, wantCode gomypal.ErrorCode) {
	t.Helper()

	apiErr := apiErrorOf(t, err)

	if apiErr.Op != wantOp || apiErr.StatusCode != wantStatus || apiErr.Code != wantCode {
		t.Errorf("api error = op %s status %d code %d, want op %s status %d code %d",
			apiErr.Op, apiErr.StatusCode, apiErr.Code, wantOp, wantStatus, wantCode)
	}

	if len(apiErr.Body) == 0 {
		t.Error("api error has no raw body")
	}
}

func assertMissingField(t *testing.T, err error, mock *mockHTTPClient) {
	t.Helper()

	if !errors.Is(err, gomypal.ErrMissingField) {
		t.Errorf("error = %v, want ErrMissingField", err)
	}

	if mock.calls != 0 {
		t.Errorf("http calls = %d, want 0: the request must be rejected before the call", mock.calls)
	}
}

type mockHTTPClient struct {
	status   int
	body     string
	err      error
	calls    int
	lastReq  *http.Request
	lastBody string
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.calls++
	m.lastReq = req

	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		m.lastBody = string(raw)
	}

	if m.err != nil {
		return nil, m.err
	}

	return &http.Response{
		StatusCode: m.status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(m.body)),
	}, nil
}

func newMockClient(t *testing.T, mock *mockHTTPClient) *gomypal.Client {
	t.Helper()

	_, keyBase64 := throwawayKey(t)

	client, err := gomypal.NewClient("https://mock.mypal.example/openapi", testAppID, keyBase64, gomypal.WithHTTPClient(mock))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return client
}

func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	return ctx
}

type sandboxT interface {
	Helper()
	Skipf(format string, args ...any)
	Fatalf(format string, args ...any)
}

func newSandboxClient(t *testing.T) *gomypal.Client {
	t.Helper()

	return sandboxClientFromEnv(t, os.Getenv, os.ReadFile)
}

func sandboxClientFromEnv(t sandboxT, getenv func(string) string, readFile func(string) ([]byte, error)) *gomypal.Client {
	t.Helper()

	baseURL := getenv(envBaseURL)
	if baseURL != "" && isProductionBaseURL(baseURL) {
		t.Fatalf("%s points at production; sandbox tests never run there", envBaseURL)
	}

	appID := getenv(envAppID)
	privateKey := getenv(envPrivateKey)

	if privateKey == "" {
		if path := getenv(envPrivateKeyFile); path != "" {
			raw, err := readFile(path)
			if err != nil {
				t.Fatalf("read the file named by %s: %v", envPrivateKeyFile, err)
			}

			privateKey = string(raw)
		}
	}

	if baseURL == "" || appID == "" || privateKey == "" {
		t.Skipf("sandbox test skipped: set %s, %s and %s or %s", envBaseURL, envAppID, envPrivateKey, envPrivateKeyFile)
	}

	client, err := gomypal.NewClient(baseURL, appID, privateKey)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return client
}

func isProductionBaseURL(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	production, _ := url.Parse(gomypal.ProductionBaseURL)

	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")

	return err != nil || host == production.Hostname() || strings.EqualFold(strings.TrimRight(baseURL, "/"), gomypal.ProductionBaseURL)
}

func sandboxEnv(t *testing.T, name string) string {
	t.Helper()

	value := os.Getenv(name)
	if value == "" {
		t.Skipf("sandbox test skipped: set %s", name)
	}

	return value
}
