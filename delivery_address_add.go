package gomypal

import (
	"context"
	"net/http"
)

// AddDeliveryAddressRequest is the input of AddDeliveryAddress.
type AddDeliveryAddressRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// CountryRegionID is DeliveryRegion.ID. Every field is required.
	CountryRegionID int64  `json:"countryRegionId"`
	Country         string `json:"country"`
	City            string `json:"city"`
	ReceiverName    string `json:"receiverName"`
	ReceiverMobile  string `json:"receiverMobile"`
	ReceiverAddress string `json:"receiverAddress"`
	PostCode        string `json:"postCode"`
}

// AddDeliveryAddressResponse is the data returned by AddDeliveryAddress.
type AddDeliveryAddressResponse struct {
	// ID is the address id to pass as DeliveryAddressID to ApplyCard.
	ID int64 `json:"id"`
}

// AddDeliveryAddress saves a mailing address for a user.
func (c *Client) AddDeliveryAddress(ctx context.Context, req *AddDeliveryAddressRequest) (*Response[AddDeliveryAddressResponse], error) {
	const op = "AddDeliveryAddress"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
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

	return call[AddDeliveryAddressResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/delivery/address/add",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
