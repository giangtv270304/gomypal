package gomypal

import (
	"context"
	"net/http"
)

// KYCCountry is one country accepted for KYC.
type KYCCountry struct {
	// Name is the country name in Chinese; EnName is the English name.
	Name   string `json:"name"`
	EnName string `json:"enName"`
	// Value is the ISO 3166-1 alpha-3 code, for example "CHN". Use it for NationCode and CountryCode.
	Value string `json:"value"`
	// ProfValue is the alpha-2 code; Code is the numeric code, for example "156".
	ProfValue string `json:"profValue"`
	Code      string `json:"code"`
	// TelephoneCode is the phone prefix without "+", for example "86".
	TelephoneCode string `json:"telephoneCode"`
	// NeedCertificate means KYC for this country also needs FileType, FileURL and FaceURL.
	NeedCertificate bool `json:"needCertificate"`
}

// ListKYCCountriesRequest is the input of ListKYCCountries.
type ListKYCCountriesRequest struct {
	// Name filters by country name. Empty returns every country.
	Name string `json:"name,omitempty"`
}

// ListKYCCountriesResponse is the data returned by ListKYCCountries.
type ListKYCCountriesResponse []KYCCountry

// ListKYCCountries returns the countries accepted for KYC. Pass nil for all countries.
func (c *Client) ListKYCCountries(ctx context.Context, req *ListKYCCountriesRequest) (*Response[ListKYCCountriesResponse], error) {
	const op = "ListKYCCountries"

	if req == nil {
		req = &ListKYCCountriesRequest{Name: ""}
	}

	return call[ListKYCCountriesResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/user/kyc/country/list",
		UID:         "",
		RequireUID:  false,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
