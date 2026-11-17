package gomypal

import (
	"context"
	"net/http"
)

// CardStatus is the lifecycle state of a card.
type CardStatus int

const (
	CardStatusInitial            CardStatus = 0
	CardStatusPendingActivation  CardStatus = 1
	CardStatusActivating         CardStatus = 2
	CardStatusNormal             CardStatus = 3
	CardStatusFrozen             CardStatus = 4
	CardStatusLossReported       CardStatus = 5
	CardStatusBeforeCancellation CardStatus = 6
	CardStatusCancelled          CardStatus = 7
	CardStatusExpired            CardStatus = 8
	CardStatusPaused             CardStatus = 9
)

// UserCard is one card of a user.
type UserCard struct {
	UserBankcardID ID         `json:"userBankcardId"`
	CardType       CardType   `json:"cardType"`
	ProductID      ID         `json:"productId"`
	CardNumber     string     `json:"cardNumber"`
	Status         CardStatus `json:"status"`
	Currency       string     `json:"currency"`
}

// ListCardsRequest is the input of ListCards.
type ListCardsRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
}

// ListCardsResponse is the data returned by ListCards.
type ListCardsResponse []UserCard

// ListCards returns the cards of a user.
func (c *Client) ListCards(ctx context.Context, req *ListCardsRequest) (*Response[ListCardsResponse], error) {
	const op = "ListCards"

	if req == nil {
		return nil, ErrNilRequest
	}

	return call[ListCardsResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/user/card/list",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        nil,
		File:        nil,
	})
}
