package gomypal

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"net/http"
)

// ErrInvalidEncryptedPIN is returned when an encrypted PIN is not base64 of an IV and whole AES blocks,
// or does not decrypt with the given key.
var ErrInvalidEncryptedPIN = errors.New("gomypal: encrypted pin cannot be decrypted")

// QueryCardPINRequest is the input of QueryCardPIN.
type QueryCardPINRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// UserBankcardID is the card id as decimal text; it is sent as a JSON string.
	UserBankcardID string `json:"userBankcardId"`
}

// QueryCardPINResponse is the data returned by QueryCardPIN.
type QueryCardPINResponse struct {
	// PIN is encrypted. Decrypt it with DecryptPIN and never log it.
	PIN string `json:"pin"`
}

// QueryCardPIN returns the encrypted PIN of a physical card. Only some card BINs support it.
func (c *Client) QueryCardPIN(ctx context.Context, req *QueryCardPINRequest) (*Response[QueryCardPINResponse], error) {
	const op = "QueryCardPIN"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == ""},
	); err != nil {
		return nil, err
	}

	return call[QueryCardPINResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/queryPin",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}

// DecryptPIN decrypts the PIN returned by QueryCardPIN (base64 of a 16-byte IV followed by AES-CBC ciphertext).
// apiAESKey is the base64 api-aesKey from the merchant dashboard.
func DecryptPIN(apiAESKey, encrypted string) (string, error) {
	block, err := pinCipher(apiAESKey)
	if err != nil {
		return "", err
	}

	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil || len(raw) < 2*aes.BlockSize || len(raw)%aes.BlockSize != 0 {
		return "", ErrInvalidEncryptedPIN
	}

	plain := make([]byte, len(raw)-aes.BlockSize)
	cipher.NewCBCDecrypter(block, raw[:aes.BlockSize]).CryptBlocks(plain, raw[aes.BlockSize:])

	padding := int(plain[len(plain)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(plain) {
		return "", ErrInvalidEncryptedPIN
	}

	for _, value := range plain[len(plain)-padding:] {
		if int(value) != padding {
			return "", ErrInvalidEncryptedPIN
		}
	}

	return string(plain[:len(plain)-padding]), nil
}

// DecryptPIN is the package DecryptPIN with the key given to WithPINKey.
func (c *Client) DecryptPIN(encrypted string) (string, error) {
	return DecryptPIN(c.pinKey, encrypted)
}
