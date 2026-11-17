package gomypal

import (
	"context"
	"net/http"
)

// ApplyCardRequest is the input of ApplyCard.
type ApplyCardRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// ProductID is CardProduct.ID.
	ProductID int64 `json:"productId"`
	// DeliveryAddressID is the id from AddDeliveryAddress. Required for a physical card.
	DeliveryAddressID int64 `json:"deliveryAddressId,omitempty"`
}

// ApplyCardResponse is the data returned by ApplyCard.
type ApplyCardResponse struct {
	UserBankcardID int64 `json:"userBankcardId"`
	// OrderNo is the order to pass to GetDeliveryInfo.
	OrderNo string `json:"orderNo"`
	// CardNo is not documented by MyPal and can be empty.
	CardNo string `json:"cardNo"`
}

// ApplyCard issues a card of a product to a user. Code 1123 (IsConflict) is a duplicate request;
// 1107 and 1122 mean KYC is missing or too low; 1120 and 1121 mean the merchant balance is too low.
func (c *Client) ApplyCard(ctx context.Context, req *ApplyCardRequest) (*Response[ApplyCardResponse], error) {
	const op = "ApplyCard"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"productId", req.ProductID == 0},
	); err != nil {
		return nil, err
	}

	return call[ApplyCardResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/apply",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
