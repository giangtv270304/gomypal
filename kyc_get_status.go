package gomypal

import (
	"context"
	"net/http"
)

// GetKYCStatusRequest is the input of GetKYCStatus.
type GetKYCStatusRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
}

// GetKYCStatusResponse is the data returned by GetKYCStatus.
type GetKYCStatusResponse struct {
	Status KYCStatus `json:"status"`
	// FailedReason is set when Status is KYCStatusFail.
	FailedReason string `json:"failedReason"`
}

// GetKYCStatus returns the KYC review state of a user.
func (c *Client) GetKYCStatus(ctx context.Context, req *GetKYCStatusRequest) (*Response[GetKYCStatusResponse], error) {
	const op = "GetKYCStatus"

	if req == nil {
		return nil, ErrNilRequest
	}

	return call[GetKYCStatusResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/user/kyc/status",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        nil,
		File:        nil,
	})
}
