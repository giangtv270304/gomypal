// Package gomypal provides a Go client for the MyPal API — user
// registration and KYC (Know Your Customer) submission/status for an
// onboarding/payments platform.
//
// Every method hangs directly off *Client: c.Register, c.SubmitKYC,
// c.CheckKYCStatus. Every request is RSA-signed with an appId + RSA
// private key pair issued by MyPal — see the signer subpackage for the
// signing mechanics ("Signature Description v2").
//
// # Quick start
//
//	client, err := gomypal.NewClient("https://api.mypal.example", appID, privateKeyPEM)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	reg, err := client.Register(ctx, "user@example.com")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	err = client.SubmitKYC(ctx, reg.UID, &gomypal.SubmitKYCRequest{ /* ... */ })
//	// ...
//	status, err := client.CheckKYCStatus(ctx, reg.UID)
package gomypal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gomypal/signer"
)

const UserAgent = "mypal-go-sdk"

var (
	// Client initialization errors.
	ErrEmptyBaseURL      = errors.New("baseURL is required")
	ErrEmptyAppID        = errors.New("appId is required")
	ErrEmptyPrivateKey   = errors.New("privateKey is required")
	ErrInvalidPrivateKey = errors.New("privateKey is not a valid RSA PKCS8 key")

	// Request lifecycle errors.
	ErrNilRequest    = errors.New("request is nil")
	ErrRequestEncode = errors.New("failed to encode request body")
	ErrRequestSign   = errors.New("failed to sign request")

	// HTTP / transport errors.
	ErrHTTPFailure      = errors.New("http request failed")
	ErrUnexpectedStatus = errors.New("unexpected http status code")
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Option func(*Client)

type Client struct {
	baseURL    string
	appID      string
	privateKey *signer.PrivateKey // parsed RSA key, see signer.ParsePrivateKey
	version    string
	logger     Logger
	debug      bool
	httpClient HTTPClient
}

func WithLogger(logger Logger) Option {
	return func(c *Client) { c.logger = logger }
}

func WithDebug(debug bool) Option {
	return func(c *Client) { c.debug = debug }
}

func WithHTTPClient(client HTTPClient) Option {
	return func(c *Client) { c.httpClient = client }
}

// WithVersion overrides the "version" signing header, default "v2".
func WithVersion(version string) Option {
	return func(c *Client) { c.version = version }
}

// NewClient constructs a Client. privateKeyRaw accepts either a bare
// base64 PKCS8 string (as MyPal's own docs show it) or a full PEM block.
func NewClient(baseURL, appID, privateKeyRaw string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, ErrEmptyBaseURL
	}
	if appID == "" {
		return nil, ErrEmptyAppID
	}
	if privateKeyRaw == "" {
		return nil, ErrEmptyPrivateKey
	}

	key, err := signer.ParsePrivateKey(privateKeyRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPrivateKey, err)
	}

	client := &Client{
		baseURL:    baseURL,
		appID:      appID,
		privateKey: key,
		version:    "v2",
		logger:     slog.Default(),
		debug:      false,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

func (c *Client) logDebug(msg string, attrs ...any) {
	if c.debug {
		c.logger.Debug(msg, attrs...)
	}
}

// paramsToMap converts a request struct into a map[string]any via its
// json tags, so the signer can canonicalize it the same way regardless
// of whether the endpoint has a body at all (nil -> empty map, matching
// an empty bodyString per the signing spec).
func paramsToMap(params any) (map[string]any, error) {
	if params == nil {
		return map[string]any{}, nil
	}
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) sign(req *request, headers http.Header, bodyMap map[string]any) error {
	signHeaders, err := signer.BuildHeaders(c.appID, c.privateKey, signer.SignParams{
		UID:     req.UID,
		Version: c.version,
		Body:    bodyMap,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRequestSign, err)
	}

	for k, v := range signHeaders {
		// uid is only an actual HTTP header on endpoints that need it
		// (Submit KYC, Check KYC Status) — Register signs an empty uid
		// but must NOT send the header at all. See doc's "Signature
		// Notes" caution block.
		if k == "uid" && v == "" {
			continue
		}
		headers.Set(k, v)
	}
	return nil
}

func (c *Client) buildRequest(req *request) error {
	if req == nil {
		return ErrNilRequest
	}

	fullURL := c.baseURL + req.Endpoint

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", UserAgent)

	bodyMap, err := paramsToMap(req.Params)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRequestEncode, err)
	}

	var bodyBytes []byte
	if req.Params != nil {
		bodyBytes, err = json.Marshal(req.Params)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrRequestEncode, err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	if err := c.sign(req, headers, bodyMap); err != nil {
		return err
	}

	c.logDebug("http request", "method", req.Method, "url", fullURL)

	req.FullURL = fullURL
	req.Header = headers
	req.Body = bytes.NewReader(bodyBytes)

	return nil
}

func (c *Client) excute(ctx context.Context, req *request) ([]byte, error) {
	if err := c.buildRequest(req); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.FullURL, req.Body)
	if err != nil {
		return nil, err
	}
	httpReq.Header = req.Header

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHTTPFailure, err)
	}
	defer func() { _ = resp.Body.Close() }()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logDebug("http response", "status", resp.StatusCode)
	c.logDebug("http response body", "body", string(responseBody))

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrUnexpectedStatus, resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}
