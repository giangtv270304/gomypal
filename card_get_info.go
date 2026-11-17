package gomypal

import (
	"context"
	"encoding/json"
	"net/http"
)

// CardInfoType says how GetCardInfo returns the card data.
type CardInfoType int

const (
	// CardInfoTypeURL: open CardPanURL to see the card. CardInfoTypePlain: CardNumber, CVV and ExpireDate are set.
	CardInfoTypeURL   CardInfoType = 0
	CardInfoTypePlain CardInfoType = 1
)

// GetCardInfoRequest is the input of GetCardInfo.
type GetCardInfoRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID            string `json:"-"`
	UserBankcardID int64  `json:"userBankcardId"`
}

// GetCardInfoResponse holds card number and CVV when InfoType is CardInfoTypePlain. Never log it.
type GetCardInfoResponse struct {
	InfoType CardInfoType `json:"infoType"`
	// Status is read from the key "status", or from "cardStatus" when "status" is absent.
	Status     CardStatus `json:"status"`
	CardNumber string     `json:"cardNumber"`
	CVV        string     `json:"cvv"`
	// ExpireDate is year-month, for example "2030-08".
	ExpireDate string `json:"expireDate"`
	CardPanURL string `json:"cardPanUrl"`
}

// UnmarshalJSON reads the documented keys and accepts "cardStatus" in place of "status".
func (r *GetCardInfoResponse) UnmarshalJSON(data []byte) error {
	type plain GetCardInfoResponse

	wire := struct {
		*plain
		Status     *CardStatus `json:"status"`
		CardStatus *CardStatus `json:"cardStatus"`
	}{plain: (*plain)(r), Status: nil, CardStatus: nil}

	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	switch {
	case wire.Status != nil:
		r.Status = *wire.Status
	case wire.CardStatus != nil:
		r.Status = *wire.CardStatus
	}

	return nil
}

// GetCardInfo returns the card state and either the card data or a URL that shows it.
func (c *Client) GetCardInfo(ctx context.Context, req *GetCardInfoRequest) (*Response[GetCardInfoResponse], error) {
	const op = "GetCardInfo"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"userBankcardId", req.UserBankcardID == 0},
	); err != nil {
		return nil, err
	}

	return call[GetCardInfoResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/info",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
