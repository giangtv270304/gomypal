package gomypal

import (
	"context"
	"net/http"
)

// SimulateReversalRequest is the input of SimulateReversal.
type SimulateReversalRequest struct {
	// UID is optional here: MyPal does not document headers for simulate endpoints.
	UID string `json:"-"`
	// TransactionID is Transaction.TransactionID.
	TransactionID string `json:"transactionId"`
	// TransactionAmount and BillAmount are sent as JSON numbers.
	TransactionAmount Decimal `json:"transactionAmount"`
	BillAmount        Decimal `json:"billAmount"`
}

// SimulateReversalResponse is the data returned by SimulateReversal.
type SimulateReversalResponse struct{}

// SimulateReversal cancels a simulated transaction. Sandbox only: it returns ErrSandboxOnly on ProductionBaseURL.
func (c *Client) SimulateReversal(ctx context.Context, req *SimulateReversalRequest) (*Response[SimulateReversalResponse], error) {
	const op = "SimulateReversal"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"transactionId", req.TransactionID == ""},
		required{"transactionAmount", req.TransactionAmount == ""},
		required{"billAmount", req.BillAmount == ""},
	); err != nil {
		return nil, err
	}

	return call[SimulateReversalResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/simulate/reversal",
		UID:         req.UID,
		RequireUID:  false,
		SandboxOnly: true,
		Body:        req,
		File:        nil,
	})
}
