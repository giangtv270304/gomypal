package gomypal

import (
	"context"
	"net/http"
)

// CardType says whether a card is virtual or physical.
type CardType string

const (
	CardTypeVirtual  CardType = "VIRTUAL"
	CardTypePhysical CardType = "PHYSICAL"
)

// CardMode says whether a card product is a normal or a share card.
type CardMode string

const (
	CardModeNormal CardMode = "NORMAL"
	CardModeShare  CardMode = "SHARE"
)

// CardProduct is a card the merchant can issue.
type CardProduct struct {
	// ID is the product id for ApplyCard and ActivateCard.
	ID             int64    `json:"id"`
	Title          string   `json:"title"`
	BankCardNature CardType `json:"bankCardNature"`
	// Ccy is the card currency.
	Ccy string `json:"ccy"`
	// ApplyFee is the issuing fee. RechargeFee is a rate between 0 and 1.
	ApplyFee       Decimal `json:"applyFee"`
	RechargeFee    Decimal `json:"rechargeFee"`
	BankcardRegion string  `json:"bankcardRegion"`
	CardBin        string  `json:"cardBin"`
	// ActiveMinLimit and RechargeMinLimit are the smallest first and later top-ups.
	ActiveMinLimit   Decimal `json:"activeMinLimit"`
	RechargeMinLimit Decimal `json:"rechargeMinLimit"`
	// CardBrand and CardMode can be empty.
	CardBrand string   `json:"cardBrand"`
	CardMode  CardMode `json:"cardMode"`
}

// ListCardProductsRequest is the input of ListCardProducts.
type ListCardProductsRequest struct{}

// ListCardProductsResponse is the data returned by ListCardProducts.
type ListCardProductsResponse []CardProduct

// ListCardProducts returns the card products of the merchant. req may be nil.
func (c *Client) ListCardProducts(ctx context.Context, _ *ListCardProductsRequest) (*Response[ListCardProductsResponse], error) {
	const op = "ListCardProducts"

	return call[ListCardProductsResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/merchant/template/list",
		UID:         "",
		RequireUID:  false,
		SandboxOnly: false,
		Body:        nil,
		File:        nil,
	})
}
