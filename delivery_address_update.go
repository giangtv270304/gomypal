package gomypal

import (
	"context"
	"net/http"
)

// UpdateDeliveryAddressRequest is the input of UpdateDeliveryAddress.
type UpdateDeliveryAddressRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// ID is the address id from AddDeliveryAddress. Every field is required. ID and CountryRegionID are sent as JSON integers.
	ID              int64  `json:"id"`
	CountryRegionID int64  `json:"countryRegionId"`
	Country         string `json:"country"`
	City            string `json:"city"`
	ReceiverName    string `json:"receiverName"`
	ReceiverMobile  string `json:"receiverMobile"`
	ReceiverAddress string `json:"receiverAddress"`
	PostCode        string `json:"postCode"`
}

// UpdateDeliveryAddressResponse is the data returned by UpdateDeliveryAddress.
type UpdateDeliveryAddressResponse struct {
	ID ID `json:"id"`
}

// UpdateDeliveryAddress replaces a saved mailing address.
func (c *Client) UpdateDeliveryAddress(ctx context.Context, req *UpdateDeliveryAddressRequest) (*Response[UpdateDeliveryAddressResponse], error) {
	const op = "UpdateDeliveryAddress"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"id", req.ID == 0},
		required{"countryRegionId", req.CountryRegionID == 0},
		required{"country", req.Country == ""},
		required{"city", req.City == ""},
		required{"receiverName", req.ReceiverName == ""},
		required{"receiverMobile", req.ReceiverMobile == ""},
		required{"receiverAddress", req.ReceiverAddress == ""},
		required{"postCode", req.PostCode == ""},
	); err != nil {
		return nil, err
	}

	return call[UpdateDeliveryAddressResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/delivery/address/update",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
