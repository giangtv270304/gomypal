package gomypal

import (
	"context"
	"net/http"
)

// CloseCardRequest is the input of CloseCard.
type CloseCardRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// UserBankcardID is the card id as decimal text; it is sent as a JSON string.
	UserBankcardID string `json:"userBankcardId"`
}

// CloseCardResponse is the data returned by CloseCard.
type CloseCardResponse struct{}

// CloseCard cancels a card. This cannot be undone.
func (c *Client) CloseCard(ctx context.Context, req *CloseCardRequest) (*Response[CloseCardResponse], error) {
	const op = "CloseCard"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == ""},
	); err != nil {
		return nil, err
	}

	return call[CloseCardResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/close",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
