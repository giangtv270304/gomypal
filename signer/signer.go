// Package signer implements MyPal's "Signature Description v2" request
// signing: RSA-SHA1 (PKCS1v15) over a canonical string built from the 5
// fixed signing headers plus the request body's non-empty fields.
//
// Two non-obvious details, verified against the doc's own worked example
// (openssl dgst -sha1 -sign reproduced the documented signature exactly):
//
//  1. The signing header set is always the same 5 keys — appId, nonce,
//     timestamp, uid, version — even for endpoints whose actual HTTP
//     headers omit uid (e.g. Register). uid signs as an empty value then.
//  2. The header block and the body block are concatenated with NO
//     separator between them (not even "&") — confirmed empirically, not
//     just read off the doc text, which is easy to misparse here.
package signer

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // required by MyPal's Signature Description v2 (SHA1WithRSA)
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrNotRSAKey = errors.New("private key is not RSA")

// PrivateKey is an alias kept so callers (and Client) don't need to
// import crypto/rsa directly just to hold a key around.
type PrivateKey = rsa.PrivateKey

// ParsePrivateKey accepts either a full PEM block or a bare base64 PKCS8
// string (MyPal's own docs show the latter — no "-----BEGIN...-----"
// armor).
func ParsePrivateKey(raw string) (*PrivateKey, error) {
	raw = strings.TrimSpace(raw)

	var der []byte
	if block, _ := pem.Decode([]byte(raw)); block != nil {
		der = block.Bytes
	} else {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("decode base64: %w", err)
		}
		der = decoded
	}

	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse pkcs8: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrNotRSAKey
	}
	return rsaKey, nil
}

type SignParams struct {
	UID     string
	Version string
	Body    map[string]any
}

// BuildHeaders generates a fresh nonce+timestamp, signs, and returns all
// 6 header values (appId, nonce, timestamp, uid, version, sign) as a
// map. Callers decide whether to actually attach an empty "uid" as an
// HTTP header (Client.sign strips it when empty).
func BuildHeaders(appID string, privateKey *PrivateKey, p SignParams) (map[string]string, error) {
	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	headerFields := map[string]string{
		"appId":     appID,
		"nonce":     nonce,
		"timestamp": timestamp,
		"uid":       p.UID,
		"version":   p.Version,
	}

	signString := BuildSignString(headerFields, p.Body)

	sig, err := Sign(privateKey, signString)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}

	headerFields["sign"] = sig
	return headerFields, nil
}

// BuildSignString reproduces the exact canonicalization from "Signature
// Description v2": the header block (all 5 keys, sorted, unfiltered)
// directly concatenated (no separator) with the body block (non-empty
// top-level fields, sorted, one level of nested-object support).
// Exported so tests can hit the documented golden vector directly with
// fixed header values, instead of only through BuildHeaders' generated
// nonce/timestamp.
func BuildSignString(headerFields map[string]string, body map[string]any) string {
	return canonicalHeaderString(headerFields) + canonicalBodyString(body)
}

func canonicalHeaderString(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+fields[k])
	}
	return strings.Join(parts, "&")
}

// canonicalBodyString renders the top-level of body as "k=v&k2=v2",
// skipping nil/"" values ("non-empty fields in the body" per the doc).
// Nested map values render one level deeper via canonicalNestedObject.
func canonicalBodyString(body map[string]any) string {
	keys := make([]string, 0, len(body))
	for k, v := range body {
		if isEmptyValue(v) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+stringifyValue(body[k]))
	}
	return strings.Join(parts, "&")
}

// canonicalNestedObject renders a nested JSON object as "{k:v,k2:v2}" —
// the doc demonstrates exactly one level of this; the implementation
// recurses generically in case a future endpoint nests deeper.
func canonicalNestedObject(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if isEmptyValue(v) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+":"+stringifyValue(m[k]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return s == ""
	}
	return false
}

// stringifyValue renders a JSON-decoded value (map[string]any's leaves
// are string/float64/bool/map[string]any/nil per encoding/json) in the
// unquoted, direct form the Java reference implementation produces
// (StringBuilder.append(value) — no JSON quoting of strings).
func stringifyValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case map[string]any:
		return canonicalNestedObject(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// Sign computes SHA1WithRSA (PKCS1v15) over data and base64-encodes it —
// the Go equivalent of the doc's Java RSAUtils.sign /
// Signature.getInstance("SHA1withRSA").
func Sign(privateKey *PrivateKey, data string) (string, error) {
	digest := sha1.Sum([]byte(data)) //nolint:gosec // required by MyPal's Signature Description v2 (SHA1WithRSA)
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA1, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func generateNonce() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 32

	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}
