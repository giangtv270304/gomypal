package gomypal

import (
	"context"
	"net/http"
)

// RegisterCustomerRequest is the input of RegisterCustomer.
type RegisterCustomerRequest struct {
	// Email must be unique per merchant.
	Email string `json:"email"`
}

// RegisterCustomerResponse is the data returned by RegisterCustomer.
type RegisterCustomerResponse struct {
	// UID is the new MyPal user id. Pass it as UID (decimal text) on later calls.
	UID int64 `json:"uid"`
}

// RegisterCustomer creates a MyPal user for an email. Code 1011 (IsConflict) means the email is already registered.
func (c *Client) RegisterCustomer(ctx context.Context, req *RegisterCustomerRequest) (*Response[RegisterCustomerResponse], error) {
	const op = "RegisterCustomer"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"email", req.Email == ""},
	); err != nil {
		return nil, err
	}

	return call[RegisterCustomerResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/user/register",
		UID:         "",
		RequireUID:  false,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
