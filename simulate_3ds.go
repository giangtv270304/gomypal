package gomypal

import (
	"context"
	"net/http"
)

// Simulate3DSRequest is the input of Simulate3DS.
type Simulate3DSRequest struct {
	// UID is optional here: MyPal does not document headers for simulate endpoints.
	UID string `json:"-"`
	// BankCardID is the card id.
	BankCardID int64 `json:"bankCardId"`
	// TransactionAmount is sent as a JSON number.
	TransactionAmount   Decimal `json:"transactionAmount"`
	TransactionCurrency string  `json:"transactionCurrency"`
}

// Simulate3DSResponse is the data returned by Simulate3DS.
type Simulate3DSResponse struct{}

// Simulate3DS triggers a "3ds" webhook for a card. Sandbox only: it returns ErrSandboxOnly on ProductionBaseURL.
func (c *Client) Simulate3DS(ctx context.Context, req *Simulate3DSRequest) (*Response[Simulate3DSResponse], error) {
	const op = "Simulate3DS"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"bankCardId", req.BankCardID == 0},
		required{"transactionAmount", req.TransactionAmount == ""},
		required{"transactionCurrency", req.TransactionCurrency == ""},
	); err != nil {
		return nil, err
	}

	return call[Simulate3DSResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/simulate/3ds",
		UID:         req.UID,
		RequireUID:  false,
		SandboxOnly: true,
		Body:        req,
		File:        nil,
	})
}
