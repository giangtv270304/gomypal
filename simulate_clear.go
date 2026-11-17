package gomypal

import (
	"context"
	"net/http"
)

// SimulateClearRequest is the input of SimulateClear.
type SimulateClearRequest struct {
	// UID is optional here: MyPal does not document headers for simulate endpoints.
	UID string `json:"-"`
	// TransactionID is Transaction.TransactionID.
	TransactionID string `json:"transactionId"`
	// TransactionAmount and BillAmount are sent as JSON numbers.
	BillAmount          Decimal `json:"billAmount"`
	TransactionAmount   Decimal `json:"transactionAmount"`
	TransactionCurrency string  `json:"transactionCurrency"`
}

// SimulateClearResponse is the data returned by SimulateClear.
type SimulateClearResponse struct{}

// SimulateClear settles a simulated transaction. Sandbox only: it returns ErrSandboxOnly on ProductionBaseURL.
func (c *Client) SimulateClear(ctx context.Context, req *SimulateClearRequest) (*Response[SimulateClearResponse], error) {
	const op = "SimulateClear"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"transactionId", req.TransactionID == ""},
		required{"billAmount", req.BillAmount == ""},
		required{"transactionAmount", req.TransactionAmount == ""},
		required{"transactionCurrency", req.TransactionCurrency == ""},
	); err != nil {
		return nil, err
	}

	return call[SimulateClearResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/simulate/clear",
		UID:         req.UID,
		RequireUID:  false,
		SandboxOnly: true,
		Body:        req,
		File:        nil,
	})
}
