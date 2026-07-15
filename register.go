package gomypal

import (
	"context"
	"encoding/json"
	"net/http"
)

// RegisterRequest is the payload for a new user's registration.
type RegisterRequest struct {
	Email string `json:"email"`
}

// RegisterResponse holds the server-generated UID — use it for every
// subsequent SubmitKYC/CheckKYCStatus call for this user.
type RegisterResponse struct {
	UID int64 `json:"uid"`
}

// Register creates a new MyPal user for email and returns its UID.
//
// Example:
//
//	resp, err := client.Register(ctx, "user@example.com")
//	if err != nil {
//		return err
//	}
//	uid := resp.UID
func (c *Client) Register(ctx context.Context, email string) (*RegisterResponse, error) {
	req := &request{
		Method:   http.MethodPost,
		Endpoint: "/user/register",
		Params:   RegisterRequest{Email: email},
	}

	raw, err := c.excute(ctx, req)
	if err != nil {
		return nil, err
	}

	resp := new(RegisterResponse)
	if err := json.Unmarshal(raw, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
