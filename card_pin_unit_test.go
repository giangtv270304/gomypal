package gomypal_test

import (
	"errors"
	"testing"

	"github.com/giangtv270304/gomypal"
)

const (
	docPINKey       = "z+7ruiMj0GottZjhNVJG2ZJsDcXqR2b3iW32+Hav34c="
	docPINEncrypted = "dsOiXZoeUUBgjw5YujQlvNzMtULGKy3vZcoLgiXgfVc="
)

func TestDecryptPIN_DocVector(t *testing.T) {
	t.Parallel()

	pin, err := gomypal.DecryptPIN(docPINKey, docPINEncrypted)
	if err != nil || pin != "123456" {
		t.Errorf("DecryptPIN = %q, %v; want 123456", pin, err)
	}
}

func TestEncryptPIN_RoundTripWithRandomIV(t *testing.T) {
	t.Parallel()

	first, err := gomypal.EncryptPIN(docPINKey, "123456")
	if err != nil {
		t.Fatalf("EncryptPIN: %v", err)
	}

	second, err := gomypal.EncryptPIN(docPINKey, "123456")
	if err != nil {
		t.Fatalf("EncryptPIN: %v", err)
	}

	if first == second || len(first) != len(docPINEncrypted) {
		t.Errorf("ciphertexts %q and %q: want different values of the documented length", first, second)
	}

	pin, err := gomypal.DecryptPIN(docPINKey, first)
	if err != nil || pin != "123456" {
		t.Errorf("DecryptPIN = %q, %v; want 123456", pin, err)
	}
}

func TestPIN_InvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := gomypal.EncryptPIN("", "123456"); !errors.Is(err, gomypal.ErrInvalidPINKey) {
		t.Errorf("empty key: error = %v, want ErrInvalidPINKey", err)
	}

	for _, shortKey := range []string{"AAAAAAAAAAAAAAAAAAAAAA==", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		if _, err := gomypal.EncryptPIN(shortKey, "123456"); !errors.Is(err, gomypal.ErrInvalidPINKey) {
			t.Errorf("key %q (not 32 bytes): error = %v, want ErrInvalidPINKey", shortKey, err)
		}
	}

	if _, err := gomypal.EncryptPIN("not-base64!", "123456"); !errors.Is(err, gomypal.ErrInvalidPINKey) {
		t.Errorf("bad key: error = %v, want ErrInvalidPINKey", err)
	}

	if _, err := gomypal.EncryptPIN(docPINKey, ""); !errors.Is(err, gomypal.ErrMissingField) {
		t.Errorf("empty pin: error = %v, want ErrMissingField", err)
	}

	for _, encrypted := range []string{"", "AAAA", "not-base64!", docPINEncrypted[:len(docPINEncrypted)-4] + "AAA="} {
		if pin, err := gomypal.DecryptPIN(docPINKey, encrypted); !errors.Is(err, gomypal.ErrInvalidEncryptedPIN) {
			t.Errorf("DecryptPIN(%q) = %q, %v; want ErrInvalidEncryptedPIN", encrypted, pin, err)
		}
	}
}

func TestClient_PINKeyOption(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)

	client, err := gomypal.NewClient("https://x.example", testAppID, keyBase64, gomypal.WithPINKey(docPINKey))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	encrypted, err := client.EncryptPIN("654321")
	if err != nil {
		t.Fatalf("EncryptPIN: %v", err)
	}

	if pin, err := client.DecryptPIN(encrypted); err != nil || pin != "654321" {
		t.Errorf("DecryptPIN = %q, %v; want 654321", pin, err)
	}

	withoutKey, err := gomypal.NewClient("https://x.example", testAppID, keyBase64)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := withoutKey.EncryptPIN("654321"); !errors.Is(err, gomypal.ErrInvalidPINKey) {
		t.Errorf("no key: error = %v, want ErrInvalidPINKey", err)
	}
}
