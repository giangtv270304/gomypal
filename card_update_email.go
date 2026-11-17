package gomypal

import (
	"context"
	"net/http"
)

// UpdateCardEmailRequest is the input of UpdateCardEmail.
type UpdateCardEmailRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// UserBankcardID is the card id as decimal text; it is sent as a JSON string.
	UserBankcardID string `json:"userBankcardId"`
	Email          string `json:"email"`
}

// UpdateCardEmailResponse is the data returned by UpdateCardEmail.
type UpdateCardEmailResponse struct{}

// UpdateCardEmail changes the email bound to a card. Only some card BINs support it (code 1111 otherwise).
func (c *Client) UpdateCardEmail(ctx context.Context, req *UpdateCardEmailRequest) (*Response[UpdateCardEmailResponse], error) {
	const op = "UpdateCardEmail"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == ""},
		required{"email", req.Email == ""},
	); err != nil {
		return nil, err
	}

	return call[UpdateCardEmailResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/update/email",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
