package gomypal

import (
	"context"
	"net/http"
)

// UpdateKYCRequest is the input of UpdateKYC.
type UpdateKYCRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	KYCInfo
}

// UpdateKYCResponse is the data returned by UpdateKYC.
type UpdateKYCResponse struct{}

// UpdateKYC replaces the user's KYC data. It takes the same fields as SubmitKYC; poll GetKYCStatus for the result.
func (c *Client) UpdateKYC(ctx context.Context, req *UpdateKYCRequest) (*Response[UpdateKYCResponse], error) {
	const op = "UpdateKYC"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op, req.requiredFields()...); err != nil {
		return nil, err
	}

	return call[UpdateKYCResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/user/kyc/update",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
