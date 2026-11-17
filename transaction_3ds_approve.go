package gomypal

import (
	"context"
	"net/http"
)

// Approve3DSRequest is the input of Approve3DS.
type Approve3DSRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// AuthID is ThreeDSEvent.AuthID from the webhook.
	AuthID string `json:"authId"`
}

// Approve3DSResponse is the data returned by Approve3DS.
type Approve3DSResponse struct{}

// Approve3DS approves a 3DS authorization announced by a "3ds" webhook with verificationType "http".
func (c *Client) Approve3DS(ctx context.Context, req *Approve3DSRequest) (*Response[Approve3DSResponse], error) {
	const op = "Approve3DS"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"authId", req.AuthID == ""},
	); err != nil {
		return nil, err
	}

	return call[Approve3DSResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/3ds/approve",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
