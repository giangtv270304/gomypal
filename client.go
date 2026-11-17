// Package gomypal is a Go client for the MyPal card API.
//
//	client, err := gomypal.NewClient(baseURL, appID, privateKey)
//	if err != nil {
//		return err
//	}
//
//	resp, err := client.RegisterCustomer(ctx, &gomypal.RegisterCustomerRequest{Email: "user@example.com"})
package gomypal

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/giangtv270304/gomypal/signer"
)

const (
	// Version is the SDK version, sent as User-Agent "gomypal/<Version>".
	Version = "0.1.1"

	// ProductionBaseURL is the live API. MyPal publishes no sandbox URL, so there is no sandbox constant:
	// pass the sandbox base URL you were given to NewClient.
	ProductionBaseURL = "https://api.cardapi.vip/openapi"

	// DefaultTimeout bounds one call, from sending the request to reading the response.
	DefaultTimeout = 30 * time.Second

	// MaxResponseBytes caps the response body the client reads. A larger body is ErrResponseTooLarge.
	MaxResponseBytes = 10 << 20

	// MaxUploadBytes caps the file UploadKYCFile reads into memory. A larger file is ErrFileTooLarge.
	MaxUploadBytes = 10 << 20

	userAgent = "gomypal/" + Version
)

var (
	ErrEmptyBaseURL = errors.New("gomypal: baseURL is required")
	// ErrInsecureBaseURL is returned for a base URL that is not https. Plain http is allowed only for
	// localhost, 127.0.0.1 and ::1.
	ErrInsecureBaseURL = errors.New("gomypal: baseURL must use https (http is allowed only on a loopback host)")
	// ErrInvalidBaseURL is returned for a base URL that cannot be parsed, has no host, or carries
	// user info, a query or a fragment.
	ErrInvalidBaseURL = errors.New("gomypal: baseURL must be scheme://host[/path] without user info, query or fragment")
	// ErrResponseTooLarge is wrapped together with ErrInvalidResponse when a response body is larger than MaxResponseBytes.
	ErrResponseTooLarge = errors.New("gomypal: response body is larger than MaxResponseBytes")
	ErrEmptyAppID       = errors.New("gomypal: appID is required")
	ErrEmptyPrivateKey  = errors.New("gomypal: privateKey is required")
	ErrNilRequest       = errors.New("gomypal: request is nil")
	// ErrMissingField is wrapped with the method and field name when a required value such as UID or an id is empty.
	ErrMissingField = errors.New("gomypal: required field is empty")
	// ErrSandboxOnly is returned by the Simulate methods when the client points at the host of ProductionBaseURL.
	ErrSandboxOnly = errors.New("gomypal: simulate endpoints are sandbox-only and are refused on the production base URL")
	// ErrInvalidResponse is wrapped when the server answered but the answer cannot be trusted: a status
	// below 400 that is not 2xx (redirects are not followed), a body that is not the {code,msg,data} envelope,
	// data that does not decode, or a body over MaxResponseBytes. For a money call the outcome is unknown.
	ErrInvalidResponse = errors.New("gomypal: invalid response")
	// ErrFileTooLarge is returned by UploadKYCFile when Content holds more than MaxUploadBytes.
	ErrFileTooLarge = errors.New("gomypal: file is larger than MaxUploadBytes")

	errUnexpectedStatus = errors.New("status is neither 2xx nor an error status")
	errNoEnvelopeCode   = errors.New("body has no envelope code")
	// ErrEmptyWebhookKey is returned when a webhook is verified without a webhook key.
	ErrEmptyWebhookKey = errors.New("gomypal: webhook key is required")
	// ErrInvalidPINKey is returned when the PIN key is missing or is not base64 of an AES key.
	ErrInvalidPINKey = errors.New("gomypal: api-aesKey must be base64 of a 32 byte AES key")
)

// HTTPClient is satisfied by *http.Client and by a test double.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Logger is satisfied by *slog.Logger.
type Logger interface {
	Debug(msg string, args ...any)
}

// Option changes an optional setting of the Client.
type Option func(*Client)

// Client calls the MyPal card API. Create one with NewClient and reuse it; it is safe for concurrent use.
type Client struct {
	baseURL    string
	production bool
	appID      string
	privateKey *rsa.PrivateKey
	webhookKey string
	pinKey     string
	httpClient HTTPClient
	logger     Logger
	timeout    time.Duration
	now        func() time.Time
	newNonce   func() (string, error)
}

// WithHTTPClient replaces the default *http.Client. DefaultTimeout (or WithTimeout) still applies to every call.
// The default client does not follow redirects; a custom client should not either, or the signed
// headers and body go to the redirect target.
func WithHTTPClient(httpClient HTTPClient) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithLogger turns on debug logs of method, path and status. The host, headers, bodies, signatures and keys are never logged.
func WithLogger(logger Logger) Option {
	return func(c *Client) { c.logger = logger }
}

// WithTimeout replaces DefaultTimeout. A value of zero or less is ignored.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithClock replaces time.Now as the source of the timestamp header (Unix milliseconds). For tests.
func WithClock(now func() time.Time) Option {
	return func(c *Client) { c.now = now }
}

// WithNonceSource replaces the random nonce (32 characters A-Z0-9). MyPal rejects a repeated nonce. For tests.
func WithNonceSource(newNonce func() (string, error)) Option {
	return func(c *Client) { c.newNonce = newNonce }
}

// WithWebhookKey sets the webhook-aesKey used by Client.VerifyWebhookSignature. Server-side only.
func WithWebhookKey(webhookKey string) Option {
	return func(c *Client) { c.webhookKey = webhookKey }
}

// WithPINKey sets the api-aesKey (base64) used by Client.EncryptPIN and Client.DecryptPIN. Server-side only.
func WithPINKey(apiAESKey string) Option {
	return func(c *Client) { c.pinKey = apiAESKey }
}

// NewClient returns a client for baseURL (ProductionBaseURL or your sandbox URL). privateKey is the merchant
// RSA key as base64 of PKCS#8 DER, as MyPal issues it; base64 PKCS#1 and PEM are accepted too.
// It fails when an argument is empty or the key cannot be parsed. Keep the private key on a server.
func NewClient(baseURL, appID, privateKey string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, ErrEmptyBaseURL
	}

	if appID == "" {
		return nil, ErrEmptyAppID
	}

	if strings.TrimSpace(privateKey) == "" {
		return nil, ErrEmptyPrivateKey
	}

	if err := checkBaseURL(baseURL); err != nil {
		return nil, err
	}

	key, err := signer.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("gomypal: privateKey: %w", err)
	}

	client := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		production: isProductionURL(baseURL),
		appID:      appID,
		privateKey: key,
		webhookKey: "",
		pinKey:     "",
		httpClient: &http.Client{CheckRedirect: doNotFollowRedirects},
		logger:     nil,
		timeout:    DefaultTimeout,
		now:        time.Now,
		newNonce:   signer.NewNonce,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

func checkBaseURL(baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return ErrInvalidBaseURL
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		host := parsed.Hostname()
		if ip := net.ParseIP(host); host == "localhost" || (ip != nil && ip.IsLoopback()) {
			return nil
		}
	}

	return ErrInsecureBaseURL
}

func isProductionURL(baseURL string) bool {
	parsed, err := url.Parse(baseURL)
	production, _ := url.Parse(ProductionBaseURL)

	return err == nil && strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".") == production.Hostname()
}

func doNotFollowRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// String returns the base URL only. The app id and keys are redacted, so a Client is safe to print.
func (c Client) String() string {
	return "gomypal.Client{baseURL: " + c.baseURL + ", appID: [redacted], keys: [redacted]}"
}

// GoString is String, so %#v does not print the app id or keys either.
func (c Client) GoString() string {
	return c.String()
}

func call[T any](ctx context.Context, c *Client, req request) (*Response[T], error) {
	resp := new(Response[T])
	if err := c.execute(ctx, req, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) execute(ctx context.Context, req request, out envelopeFiller) error {
	if req.SandboxOnly && c.production {
		return fmt.Errorf("gomypal: %s: %w", req.Op, ErrSandboxOnly)
	}

	if req.RequireUID && req.UID == "" {
		return fmt.Errorf("gomypal: %s: uid: %w", req.Op, ErrMissingField)
	}

	body, signedBody, contentType, err := encodeBody(req)
	if err != nil {
		return err
	}

	nonce, err := c.newNonce()
	if err != nil {
		return fmt.Errorf("gomypal: %s: nonce: %w", req.Op, err)
	}

	// The signature covers the exact bytes sent: sign after encoding, never re-encode.
	signed, err := signer.Headers(c.privateKey, signer.Params{
		AppID:     c.appID,
		Nonce:     nonce,
		Timestamp: c.now(),
		UID:       req.UID,
		Version:   signer.SignatureVersion,
		Body:      signedBody,
	})
	if err != nil {
		return fmt.Errorf("gomypal: %s: sign request: %w", req.Op, err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	fullURL := c.baseURL + req.Path

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gomypal: %s: build request: %w", req.Op, err)
	}

	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", userAgent)
	httpReq.Header.Set("Content-Type", contentType)

	for key, value := range signed {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gomypal: %s: send %s %s: %w", req.Op, req.Method, req.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("gomypal: %s: read response of %s %s: %w", req.Op, req.Method, req.Path, err)
	}

	if len(raw) > MaxResponseBytes {
		return fmt.Errorf("gomypal: %s: response of %s %s (status %d): %w: %w",
			req.Op, req.Method, req.Path, resp.StatusCode, ErrInvalidResponse, ErrResponseTooLarge)
	}

	if c.logger != nil {
		c.logger.Debug("gomypal request", "method", req.Method, "path", req.Path, "status", resp.StatusCode)
	}

	return decodeResponse(req, resp.StatusCode, raw, out)
}

func encodeBody(req request) (body, signedBody []byte, contentType string, err error) {
	if req.File != nil {
		var buffer bytes.Buffer

		writer := multipart.NewWriter(&buffer)

		part, err := writer.CreateFormFile(req.File.FieldName, req.File.FileName)
		if err != nil {
			return nil, nil, "", fmt.Errorf("gomypal: %s: build multipart body: %w", req.Op, err)
		}

		copied, err := io.Copy(part, io.LimitReader(req.File.Content, MaxUploadBytes+1))
		if err != nil {
			return nil, nil, "", fmt.Errorf("gomypal: %s: read file content: %w", req.Op, err)
		}

		if copied > MaxUploadBytes {
			return nil, nil, "", fmt.Errorf("gomypal: %s: %w", req.Op, ErrFileTooLarge)
		}

		if err := writer.Close(); err != nil {
			return nil, nil, "", fmt.Errorf("gomypal: %s: build multipart body: %w", req.Op, err)
		}

		return buffer.Bytes(), nil, writer.FormDataContentType(), nil
	}

	if req.Body == nil {
		return []byte("{}"), []byte("{}"), "application/json", nil
	}

	var buffer bytes.Buffer

	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(req.Body); err != nil {
		return nil, nil, "", fmt.Errorf("gomypal: %s: encode request body: %w", req.Op, err)
	}

	encoded := bytes.TrimRight(buffer.Bytes(), "\n")

	return encoded, encoded, "application/json", nil
}

func decodeResponse(req request, statusCode int, raw []byte, out envelopeFiller) error {
	if statusCode >= http.StatusBadRequest {
		return newAPIError(req.Op, statusCode, raw)
	}

	invalid := func(cause error) error {
		return fmt.Errorf("gomypal: %s: response of %s %s (status %d): %w: %w",
			req.Op, req.Method, req.Path, statusCode, ErrInvalidResponse, cause)
	}

	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return invalid(errUnexpectedStatus)
	}

	var envelope struct {
		Code *ErrorCode      `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(raw, &envelope); err != nil {
		return invalid(err)
	}

	if envelope.Code == nil {
		return invalid(errNoEnvelopeCode)
	}

	if *envelope.Code != CodeSuccess {
		return newAPIError(req.Op, statusCode, raw)
	}

	if err := out.fill(envelope.Msg, envelope.Data); err != nil {
		return invalid(err)
	}

	return nil
}
