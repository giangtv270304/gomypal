package gomypal

import (
	"context"
	"net/http"
)

// DeliveryStatus says whether a physical card was mailed.
type DeliveryStatus string

const (
	DeliveryStatusNotDelivery DeliveryStatus = "NOT_DELIVERY"
	DeliveryStatusDelivery    DeliveryStatus = "DELIVERY"
)

// GetDeliveryInfoRequest is the input of GetDeliveryInfo.
type GetDeliveryInfoRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// OrderNo is ApplyCardResponse.OrderNo.
	OrderNo string `json:"orderNo"`
}

// GetDeliveryInfoResponse is the data returned by GetDeliveryInfo.
type GetDeliveryInfoResponse struct {
	DeliveryStatus DeliveryStatus `json:"deliveryStatus"`
	// DeliveryNo is the tracking number.
	DeliveryNo string `json:"deliveryNo"`
}

// GetDeliveryInfo returns the mailing state of a physical card order. Codes 1204 and 1205: order not found or not this user's.
func (c *Client) GetDeliveryInfo(ctx context.Context, req *GetDeliveryInfoRequest) (*Response[GetDeliveryInfoResponse], error) {
	const op = "GetDeliveryInfo"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"orderNo", req.OrderNo == ""},
	); err != nil {
		return nil, err
	}

	return call[GetDeliveryInfoResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/delivery/info",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
