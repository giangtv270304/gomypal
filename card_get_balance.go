package gomypal

import (
	"context"
	"net/http"
)

// GetCardBalanceRequest is the input of GetCardBalance.
type GetCardBalanceRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID            string `json:"-"`
	UserBankcardID int64  `json:"userBankcardId"`
}

// GetCardBalanceResponse is the data returned by GetCardBalance.
type GetCardBalanceResponse struct {
	Balance  Decimal `json:"balance"`
	Currency string  `json:"currency"`
	Symbol   string  `json:"symbol"`
	// CardLimit is empty when MyPal does not send it.
	CardLimit Decimal `json:"cardLimit"`
}

// GetCardBalance returns the balance of a card.
func (c *Client) GetCardBalance(ctx context.Context, req *GetCardBalanceRequest) (*Response[GetCardBalanceResponse], error) {
	const op = "GetCardBalance"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == 0},
	); err != nil {
		return nil, err
	}

	return call[GetCardBalanceResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/balance",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
