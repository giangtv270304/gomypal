package gomypal

import (
	"context"
	"net/http"
)

// ActivateCardRequest is the input of ActivateCard.
type ActivateCardRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID          string `json:"-"`
	ProductID    int64  `json:"productId"`
	MobilePrefix string `json:"mobilePrefix"`
	Mobile       string `json:"mobile"`
	// CountryCode is ISO 3166-1 alpha-3.
	CountryCode string `json:"countryCode"`
	Address     string `json:"address"`
	Address2    string `json:"address2,omitempty"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostCode    string `json:"postCode"`
	// CardNo and VerifyCode: at least one is required.
	CardNo     string `json:"cardNo,omitempty"`
	VerifyCode string `json:"verifyCode,omitempty"`
	CVV        string `json:"cvv,omitempty"`
	// ExpDate is month/year, for example "08/28".
	ExpDate string `json:"expDate,omitempty"`
}

// ActivateCardResponse is the data returned by ActivateCard.
type ActivateCardResponse struct {
	UserBankcardID ID `json:"userBankcardId"`
}

// ActivateCard binds and activates a physical card for a user. It fails before the call when
// both CardNo and VerifyCode are empty.
func (c *Client) ActivateCard(ctx context.Context, req *ActivateCardRequest) (*Response[ActivateCardResponse], error) {
	const op = "ActivateCard"

	if req == nil {
		return nil, ErrNilRequest
	}

	if req.CardNo == "" && req.VerifyCode == "" {
		return nil, checkRequired(op, required{"cardNo or verifyCode", true})
	}

	if err := checkRequired(op,
		required{"productId", req.ProductID == 0},
		required{"mobilePrefix", req.MobilePrefix == ""},
		required{"mobile", req.Mobile == ""},
		required{"countryCode", req.CountryCode == ""},
		required{"address", req.Address == ""},
		required{"city", req.City == ""},
		required{"state", req.State == ""},
		required{"postCode", req.PostCode == ""},
	); err != nil {
		return nil, err
	}

	return call[ActivateCardResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/active",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
