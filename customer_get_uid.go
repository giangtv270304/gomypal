package gomypal

import (
	"context"
	"net/http"
)

// GetCustomerUIDRequest is the input of GetCustomerUID.
type GetCustomerUIDRequest struct {
	Email string `json:"email"`
}

// GetCustomerUIDResponse is the data returned by GetCustomerUID.
type GetCustomerUIDResponse struct {
	UID int64 `json:"uid"`
}

// GetCustomerUID returns the uid registered for an email. Code 1007 (IsNotFound) means the email is not registered.
func (c *Client) GetCustomerUID(ctx context.Context, req *GetCustomerUIDRequest) (*Response[GetCustomerUIDResponse], error) {
	const op = "GetCustomerUID"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"email", req.Email == ""},
	); err != nil {
		return nil, err
	}

	return call[GetCustomerUIDResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/user/uid",
		UID:         "",
		RequireUID:  false,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
