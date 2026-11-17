// Package signer builds the "Signature Description v2" headers for one MyPal request:
// RSASSA-PKCS1-v1_5 with SHA-1 over a canonical string, base64 encoded.
package signer

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // MyPal's Signature Description v2 is SHA1WithRSA.
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	HeaderAppID     = "appId"
	HeaderNonce     = "nonce"
	HeaderTimestamp = "timestamp"
	HeaderUID       = "uid"
	HeaderVersion   = "version"
	HeaderSign      = "sign"

	// SignatureVersion is the value of the version header.
	SignatureVersion = "v2"

	nonceLength  = 32
	nonceCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var (
	// ErrInvalidPrivateKey is returned for a key that is not base64 or PEM of an RSA PKCS#8 or PKCS#1 key.
	// The key text is never part of the error.
	ErrInvalidPrivateKey = errors.New("signer: private key is not a base64 or PEM encoded RSA key (PKCS#8 or PKCS#1)")
	// ErrBodyNotObject is returned when the body to sign is not a JSON object.
	ErrBodyNotObject = errors.New("signer: body is not a JSON object")
	// ErrArrayNotSupported is returned for a body that holds an array: MyPal does not document its canonical form.
	ErrArrayNotSupported = errors.New("signer: arrays in the request body cannot be signed")
)

// Params is everything the signature covers. Body is the exact JSON bytes sent on the wire;
// leave it nil for a request without a JSON body (multipart upload).
type Params struct {
	AppID     string
	Nonce     string
	Timestamp time.Time
	UID       string
	Version   string
	Body      []byte
}

// ParsePrivateKey reads an RSA private key given as base64 of PKCS#8 DER (the form in the MyPal docs),
// base64 of PKCS#1 DER, or a PEM block of either.
func ParsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	raw = strings.TrimSpace(raw)

	var der []byte

	if block, _ := pem.Decode([]byte(raw)); block != nil {
		der = block.Bytes
	} else {
		compact := strings.Join(strings.Fields(raw), "")

		decoded, err := base64.StdEncoding.DecodeString(compact)
		if err != nil {
			return nil, ErrInvalidPrivateKey
		}

		der = decoded
	}

	if parsed, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if key, ok := parsed.(*rsa.PrivateKey); ok {
			return key, nil
		}

		return nil, ErrInvalidPrivateKey
	}

	key, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return nil, ErrInvalidPrivateKey
	}

	return key, nil
}

// NewNonce returns 32 random characters from A-Z0-9.
func NewNonce() (string, error) {
	const acceptBelow = 252

	nonce := make([]byte, 0, nonceLength)
	buffer := make([]byte, nonceLength)

	for len(nonce) < nonceLength {
		if _, err := rand.Read(buffer); err != nil {
			return "", fmt.Errorf("signer: read random bytes: %w", err)
		}

		for _, value := range buffer {
			if value < acceptBelow && len(nonce) < nonceLength {
				nonce = append(nonce, nonceCharset[int(value)%len(nonceCharset)])
			}
		}
	}

	return string(nonce), nil
}

// Canonical returns the string that gets signed: the five header pairs sorted by key, joined with "&",
// followed with no separator by the body pairs. Body values that are null or "" are skipped.
func Canonical(params Params) (string, error) {
	header := HeaderAppID + "=" + params.AppID +
		"&" + HeaderNonce + "=" + params.Nonce +
		"&" + HeaderTimestamp + "=" + timestampText(params.Timestamp) +
		"&" + HeaderUID + "=" + params.UID +
		"&" + HeaderVersion + "=" + params.Version

	body, err := canonicalBody(params.Body)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(header+body, `"`, ""), nil
}

// Sign returns the standard base64 RSA-SHA1 (PKCS#1 v1.5) signature of message.
func Sign(privateKey *rsa.PrivateKey, message string) (string, error) {
	digest := sha1.Sum([]byte(message)) //nolint:gosec // MyPal's Signature Description v2 is SHA1WithRSA.

	signature, err := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA1, digest[:])
	if err != nil {
		return "", fmt.Errorf("signer: rsa sign: %w", err)
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// Headers returns the six signing headers to add to the request. The uid header is present
// with an empty value when params.UID is empty.
func Headers(privateKey *rsa.PrivateKey, params Params) (map[string]string, error) {
	canonical, err := Canonical(params)
	if err != nil {
		return nil, err
	}

	signature, err := Sign(privateKey, canonical)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		HeaderAppID:     params.AppID,
		HeaderNonce:     params.Nonce,
		HeaderTimestamp: timestampText(params.Timestamp),
		HeaderUID:       params.UID,
		HeaderVersion:   params.Version,
		HeaderSign:      signature,
	}, nil
}

func timestampText(timestamp time.Time) string {
	return strconv.FormatInt(timestamp.UnixMilli(), 10)
}

func canonicalBody(body []byte) (string, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return "", nil
	}

	fields, err := decodeObject(body)
	if err != nil {
		return "", err
	}

	delete(fields, HeaderSign)

	return joinFields(fields, "=", "&")
}

func decodeObject(raw []byte) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, ErrBodyNotObject
	}

	return fields, nil
}

func joinFields(fields map[string]json.RawMessage, keyValueSeparator, pairSeparator string) (string, error) {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))

	for _, key := range keys {
		text, skip, err := valueText(fields[key])
		if err != nil {
			return "", err
		}

		if skip {
			continue
		}

		pairs = append(pairs, key+keyValueSeparator+text)
	}

	return strings.Join(pairs, pairSeparator), nil
}

func valueText(raw json.RawMessage) (text string, skip bool, err error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return "", true, nil
	}

	switch trimmed[0] {
	case '"':
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return "", false, ErrBodyNotObject
		}

		return value, value == "", nil
	case '{':
		fields, err := decodeObject(trimmed)
		if err != nil {
			return "", false, err
		}

		nested, err := joinFields(fields, ":", ",")
		if err != nil {
			return "", false, err
		}

		return "{" + nested + "}", false, nil
	case '[':
		return "", false, ErrArrayNotSupported
	default:
		return string(trimmed), false, nil
	}
}
