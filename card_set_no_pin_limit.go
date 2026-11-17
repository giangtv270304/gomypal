package gomypal

import (
	"context"
	"net/http"
)

// SetCardNoPINLimitRequest is the input of SetCardNoPINLimit.
type SetCardNoPINLimitRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID            string `json:"-"`
	UserBankcardID int64  `json:"userBankcardId"`
	// NoPinLimit is sent as a JSON number. MyPal does not document its currency.
	NoPinLimit Decimal `json:"noPinLimit"`
}

// SetCardNoPINLimitResponse is the data returned by SetCardNoPINLimit.
type SetCardNoPINLimitResponse struct{}

// SetCardNoPINLimit sets the amount below which a physical card needs no PIN.
func (c *Client) SetCardNoPINLimit(ctx context.Context, req *SetCardNoPINLimitRequest) (*Response[SetCardNoPINLimitResponse], error) {
	const op = "SetCardNoPINLimit"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == 0},
		required{"noPinLimit", req.NoPinLimit == ""},
	); err != nil {
		return nil, err
	}

	return call[SetCardNoPINLimitResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/setNoPinLimit",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
