package gomypal

import (
	"context"
	"net/http"
)

// Reject3DSRequest is the input of Reject3DS.
type Reject3DSRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// AuthID is ThreeDSEvent.AuthID from the webhook.
	AuthID string `json:"authId"`
}

// Reject3DSResponse is the data returned by Reject3DS.
type Reject3DSResponse struct{}

// Reject3DS rejects a 3DS authorization announced by a "3ds" webhook with verificationType "http".
func (c *Client) Reject3DS(ctx context.Context, req *Reject3DSRequest) (*Response[Reject3DSResponse], error) {
	const op = "Reject3DS"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"authId", req.AuthID == ""},
	); err != nil {
		return nil, err
	}

	return call[Reject3DSResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/3ds/reject",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
