package gomypal

import (
	"context"
	"net/http"
)

const maxCardAmountDecimals = 2

// RechargeCardRequest is the input of RechargeCard.
type RechargeCardRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID            string `json:"-"`
	UserBankcardID int64  `json:"userBankcardId"`
	// Amount must be above 0 with at most 2 decimals; it is sent as a JSON number.
	// Pass it without trailing zeros ("10.5", not "10.50").
	Amount Decimal `json:"amount"`
	// RequestOrderID is your unique id for this order. Sending it again gives code 1123 instead of a second order,
	// so keep the same id when you resend after an unknown outcome.
	RequestOrderID string `json:"requestOrderId"`
}

// RechargeCardResponse is the data returned by RechargeCard.
type RechargeCardResponse struct{}

// RechargeCard tops up a card from the merchant wallet. Success only means the order was accepted: wait for the
// cardRechargeResult webhook or poll GetCardBalanceChangeOrder. An error that is neither an *APIError nor a
// validation error (ErrMissingField, ErrInvalidDecimal, ErrNilRequest) leaves the outcome unknown: query the order,
// or resend with the same RequestOrderID, never with a new one.
func (c *Client) RechargeCard(ctx context.Context, req *RechargeCardRequest) (*Response[RechargeCardResponse], error) {
	const op = "RechargeCard"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == 0},
		required{"amount", req.Amount == ""},
		required{"requestOrderId", req.RequestOrderID == ""},
	); err != nil {
		return nil, err
	}

	if err := req.Amount.checkPositiveAmount(op, "amount", maxCardAmountDecimals); err != nil {
		return nil, err
	}

	return call[RechargeCardResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/recharge",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
