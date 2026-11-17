package gomypal

import (
	"context"
	"net/http"
)

// AccountType says whether a merchant account is the main or a sub account.
type AccountType string

const (
	AccountTypeMain AccountType = "MAIN"
	AccountTypeSub  AccountType = "SUB"
)

// MerchantAsset is the balance of one merchant account in one currency.
type MerchantAsset struct {
	Currency        string      `json:"currency"`
	AvailableAmount Decimal     `json:"availableAmount"`
	FrozenAmount    Decimal     `json:"frozenAmount"`
	AccountID       ID          `json:"accountId"`
	AccountType     AccountType `json:"accountType"`
}

// GetMerchantAssetsRequest is the input of GetMerchantAssets.
type GetMerchantAssetsRequest struct{}

// GetMerchantAssetsResponse is the data returned by GetMerchantAssets.
type GetMerchantAssetsResponse []MerchantAsset

// GetMerchantAssets returns the merchant wallet balances. req may be nil.
func (c *Client) GetMerchantAssets(ctx context.Context, _ *GetMerchantAssetsRequest) (*Response[GetMerchantAssetsResponse], error) {
	const op = "GetMerchantAssets"

	return call[GetMerchantAssetsResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/merchant/balance",
		UID:         "",
		RequireUID:  false,
		SandboxOnly: false,
		Body:        nil,
		File:        nil,
	})
}
