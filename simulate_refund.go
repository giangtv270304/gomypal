package gomypal

import (
	"context"
	"net/http"
)

// SimulateRefundRequest is the input of SimulateRefund.
type SimulateRefundRequest struct {
	// UID is optional here: MyPal does not document headers for simulate endpoints.
	UID string `json:"-"`
	// TransactionID is Transaction.TransactionID.
	TransactionID string `json:"transactionId"`
	// TransactionAmount and BillAmount are sent as JSON numbers.
	TransactionAmount Decimal `json:"transactionAmount"`
	BillAmount        Decimal `json:"billAmount"`
}

// SimulateRefundResponse is the data returned by SimulateRefund.
type SimulateRefundResponse struct{}

// SimulateRefund refunds a simulated transaction. Sandbox only: it returns ErrSandboxOnly on ProductionBaseURL.
func (c *Client) SimulateRefund(ctx context.Context, req *SimulateRefundRequest) (*Response[SimulateRefundResponse], error) {
	const op = "SimulateRefund"

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

	return call[SimulateRefundResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/simulate/refund",
		UID:         req.UID,
		RequireUID:  false,
		SandboxOnly: true,
		Body:        req,
		File:        nil,
	})
}
