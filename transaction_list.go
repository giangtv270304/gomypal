package gomypal

import (
	"context"
	"net/http"
)

// ListTransactionsRequest is the input of ListTransactions.
type ListTransactionsRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID            string `json:"-"`
	UserBankcardID int64  `json:"userBankcardId"`
	// PageSize defaults to 10 and PageNum to the first page when zero.
	PageSize int64 `json:"pageSize,omitempty"`
	PageNum  int64 `json:"pageNum,omitempty"`
	// StartTime and EndTime are Unix milliseconds; zero means no bound.
	StartTime int64 `json:"startTime,omitempty"`
	EndTime   int64 `json:"endTime,omitempty"`
}

// ListTransactionsResponse is the data returned by ListTransactions.
type ListTransactionsResponse []Transaction

// ListTransactions returns one page of a card's transactions. MyPal sends no total: a short page is the last one.
func (c *Client) ListTransactions(ctx context.Context, req *ListTransactionsRequest) (*Response[ListTransactionsResponse], error) {
	const op = "ListTransactions"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == 0},
	); err != nil {
		return nil, err
	}

	return call[ListTransactionsResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/trade/details",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
