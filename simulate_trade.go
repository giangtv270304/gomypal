package gomypal

import (
	"context"
	"net/http"
)

// SimulateTradeRequest is the input of SimulateTrade.
type SimulateTradeRequest struct {
	// UID is optional here: MyPal does not document headers for simulate endpoints.
	UID string `json:"-"`
	// BankCardID is the card id.
	BankCardID int64 `json:"bankCardId"`
	// TransactionAmount and BillAmount are sent as JSON numbers.
	TransactionAmount   Decimal `json:"transactionAmount"`
	TransactionCurrency string  `json:"transactionCurrency"`
	BillAmount          Decimal `json:"billAmount"`
}

// SimulateTradeResponse is the data returned by SimulateTrade.
type SimulateTradeResponse struct{}

// SimulateTrade creates a test card authorization. The response is empty: find the transaction with ListTransactions.
// Sandbox only: it returns ErrSandboxOnly on ProductionBaseURL.
func (c *Client) SimulateTrade(ctx context.Context, req *SimulateTradeRequest) (*Response[SimulateTradeResponse], error) {
	const op = "SimulateTrade"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"bankCardId", req.BankCardID == 0},
		required{"transactionAmount", req.TransactionAmount == ""},
		required{"transactionCurrency", req.TransactionCurrency == ""},
		required{"billAmount", req.BillAmount == ""},
	); err != nil {
		return nil, err
	}

	return call[SimulateTradeResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/simulate/auth",
		UID:         req.UID,
		RequireUID:  false,
		SandboxOnly: true,
		Body:        req,
		File:        nil,
	})
}
