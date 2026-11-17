package gomypal

import (
	"context"
	"net/http"
)

// CanActivateCardRequest is the input of CanActivateCard.
type CanActivateCardRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// CardNo and VerifyCode: at least one is required.
	CardNo     string `json:"cardNo,omitempty"`
	VerifyCode string `json:"verifyCode,omitempty"`
	CVV        string `json:"cvv,omitempty"`
	// ExpDate is month/year, for example "08/28".
	ExpDate string `json:"expDate,omitempty"`
}

// CanActivateCardResponse is the data returned by CanActivateCard.
type CanActivateCardResponse struct {
	CanActive bool `json:"canActive"`
}

// CanActivateCard checks whether a physical card can be activated. It fails before the call when
// both CardNo and VerifyCode are empty.
func (c *Client) CanActivateCard(ctx context.Context, req *CanActivateCardRequest) (*Response[CanActivateCardResponse], error) {
	const op = "CanActivateCard"

	if req == nil {
		return nil, ErrNilRequest
	}

	if req.CardNo == "" && req.VerifyCode == "" {
		return nil, checkRequired(op, required{"cardNo or verifyCode", true})
	}

	return call[CanActivateCardResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/canActive",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
