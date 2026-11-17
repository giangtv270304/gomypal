package gomypal

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
)

const pinKeyBytes = 32

// SetCardPINRequest is the input of SetCardPIN.
type SetCardPINRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID            string `json:"-"`
	UserBankcardID int64  `json:"userBankcardId"`
	// PIN is the encrypted PIN from EncryptPIN, never the plain digits.
	PIN string `json:"pin"`
}

// SetCardPINResponse is the data returned by SetCardPIN.
type SetCardPINResponse struct{}

// SetCardPIN sets the PIN of a physical card. Encrypt the PIN with EncryptPIN first.
func (c *Client) SetCardPIN(ctx context.Context, req *SetCardPINRequest) (*Response[SetCardPINResponse], error) {
	const op = "SetCardPIN"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == 0},
		required{"pin", req.PIN == ""},
	); err != nil {
		return nil, err
	}

	return call[SetCardPINResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/setPin",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}

// EncryptPIN encrypts a card PIN for SetCardPIN: AES-CBC with PKCS#5 padding and a random IV,
// returned as base64 of IV followed by the ciphertext. apiAESKey is the base64 api-aesKey (32 bytes) from the merchant dashboard.
func EncryptPIN(apiAESKey, pin string) (string, error) {
	block, err := pinCipher(apiAESKey)
	if err != nil {
		return "", err
	}

	if pin == "" {
		return "", fmt.Errorf("gomypal: EncryptPIN: pin: %w", ErrMissingField)
	}

	padding := aes.BlockSize - len(pin)%aes.BlockSize
	plain := append([]byte(pin), bytes.Repeat([]byte{byte(padding)}, padding)...)

	out := make([]byte, aes.BlockSize+len(plain))
	if _, err := rand.Read(out[:aes.BlockSize]); err != nil {
		return "", fmt.Errorf("gomypal: EncryptPIN: read random iv: %w", err)
	}

	cipher.NewCBCEncrypter(block, out[:aes.BlockSize]).CryptBlocks(out[aes.BlockSize:], plain)

	return base64.StdEncoding.EncodeToString(out), nil
}

// EncryptPIN is the package EncryptPIN with the key given to WithPINKey.
func (c *Client) EncryptPIN(pin string) (string, error) {
	return EncryptPIN(c.pinKey, pin)
}

func pinCipher(apiAESKey string) (cipher.Block, error) {
	key, err := base64.StdEncoding.DecodeString(apiAESKey)
	if err != nil || len(key) != pinKeyBytes {
		return nil, ErrInvalidPINKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrInvalidPINKey
	}

	return block, nil
}
