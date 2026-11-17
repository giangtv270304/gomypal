package gomypal

import (
	"context"
	"net/http"
)

// Locale selects the language of region names.
type Locale string

const (
	LocaleSimplifiedChinese  Locale = "zh_CN"
	LocaleTraditionalChinese Locale = "zh_HK"
	LocaleEnglish            Locale = "en_US"
	LocaleKorean             Locale = "ko_KR"
)

// DeliveryRegion is a country or region physical cards can be mailed to.
type DeliveryRegion struct {
	// ID is the CountryRegionID for AddDeliveryAddress.
	ID              ID     `json:"id"`
	CountryRegion   string `json:"countryRegion"`
	CountryRegionZh string `json:"countryRegionZh"`
	NationalCode    string `json:"nationalCode"`
	// Price is the postage. MyPal does not document its currency.
	Price Decimal `json:"price"`
}

// ListDeliveryRegionsRequest is the input of ListDeliveryRegions.
type ListDeliveryRegionsRequest struct {
	// Local is the language of the names. Required. The JSON key is "local".
	Local Locale `json:"local"`
}

// ListDeliveryRegionsResponse is the data returned by ListDeliveryRegions.
type ListDeliveryRegionsResponse []DeliveryRegion

// ListDeliveryRegions returns the regions physical cards can be mailed to.
func (c *Client) ListDeliveryRegions(ctx context.Context, req *ListDeliveryRegionsRequest) (*Response[ListDeliveryRegionsResponse], error) {
	const op = "ListDeliveryRegions"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"local", req.Local == ""},
	); err != nil {
		return nil, err
	}

	return call[ListDeliveryRegionsResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/delivery/region",
		UID:         "",
		RequireUID:  false,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
