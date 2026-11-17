package gomypal

import (
	"context"
	"net/http"
)

// UpdateCardStatusRequest is the input of UpdateCardStatus.
type UpdateCardStatusRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID            string `json:"-"`
	UserBankcardID int64  `json:"userBankcardId"`
	// Enable true unfreezes the card, false freezes it.
	Enable bool `json:"enable"`
}

// UpdateCardStatusResponse is the data returned by UpdateCardStatus.
type UpdateCardStatusResponse struct{}

// UpdateCardStatus freezes or unfreezes a card. MyPal rate-limits two writes on one card within about a second.
func (c *Client) UpdateCardStatus(ctx context.Context, req *UpdateCardStatusRequest) (*Response[UpdateCardStatusResponse], error) {
	const op = "UpdateCardStatus"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == 0},
	); err != nil {
		return nil, err
	}

	return call[UpdateCardStatusResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/update/status",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
