package gomypal

import (
	"context"
	"net/http"
)

// BalanceChangeType selects the kind of order to query.
type BalanceChangeType string

const (
	BalanceChangeTypeRecharge BalanceChangeType = "RECHARGE"
	BalanceChangeTypeWithdraw BalanceChangeType = "WITHDRAW"
)

// OrderStatus is the state of a recharge or withdraw order. Webhooks use only SUCCESS and FAILURE.
type OrderStatus string

const (
	OrderStatusPending OrderStatus = "PENDING"
	OrderStatusSuccess OrderStatus = "SUCCESS"
	OrderStatusFailure OrderStatus = "FAILURE"
)

// GetCardBalanceChangeOrderRequest is the input of GetCardBalanceChangeOrder.
type GetCardBalanceChangeOrderRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// RequestOrderID is the id you sent to RechargeCard or WithdrawCard.
	RequestOrderID string            `json:"requestOrderId"`
	Type           BalanceChangeType `json:"type"`
}

// GetCardBalanceChangeOrderResponse is empty (zero values) when MyPal returns null data.
type GetCardBalanceChangeOrderResponse struct {
	RequestOrderID string      `json:"requestOrderId"`
	Status         OrderStatus `json:"status"`
}

// GetCardBalanceChangeOrder returns the state of a recharge or withdraw order.
// Code 1124 (IsNotFound) means MyPal never received the order.
func (c *Client) GetCardBalanceChangeOrder(ctx context.Context, req *GetCardBalanceChangeOrderRequest) (*Response[GetCardBalanceChangeOrderResponse], error) {
	const op = "GetCardBalanceChangeOrder"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"requestOrderId", req.RequestOrderID == ""},
		required{"type", req.Type == ""},
	); err != nil {
		return nil, err
	}

	return call[GetCardBalanceChangeOrderResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/balance/change/order",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
