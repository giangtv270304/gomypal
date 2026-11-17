package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestListCards_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/user/card/list",
		uid:      testUID,
		body:     `{}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":[{"userBankcardId":100168,"cardType":"VIRTUAL","productId":"100004","cardNumber":"428852******1234","status":3,"currency":"USD"},{"userBankcardId":"100169","cardType":"METAL","productId":100005,"cardNumber":"","status":42,"currency":"USD"}]}`,
	})

	resp, err := client.ListCards(testContext(t), &gomypal.ListCardsRequest{UID: testUID})
	if err != nil {
		t.Fatalf("ListCards: %v", err)
	}

	if got, want := len(resp.Data), 2; got != want {
		t.Errorf("len(resp.Data) = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].UserBankcardID, gomypal.ID(100168); got != want {
		t.Errorf("resp.Data[0].UserBankcardID = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].ProductID, gomypal.ID(100004); got != want {
		t.Errorf("resp.Data[0].ProductID = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].Status, gomypal.CardStatusNormal; got != want {
		t.Errorf("resp.Data[0].Status = %v, want %v", got, want)
	}

	if got, want := resp.Data[1].UserBankcardID, gomypal.ID(100169); got != want {
		t.Errorf("resp.Data[1].UserBankcardID = %v, want %v", got, want)
	}

	if got, want := resp.Data[1].CardType, gomypal.CardType("METAL"); got != want {
		t.Errorf("resp.Data[1].CardType = %v, want %v", got, want)
	}

	if got, want := resp.Data[1].Status, gomypal.CardStatus(42); got != want {
		t.Errorf("resp.Data[1].Status = %v, want %v", got, want)
	}
}

func TestListCards_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/user/card/list",
		uid:      testUID,
		body:     `{}`,
		status:   http.StatusOK,
		response: `{"code":1013,"msg":"no card in possession"}`,
	})

	_, err := client.ListCards(testContext(t), &gomypal.ListCardsRequest{UID: testUID})

	assertAPIError(t, err, "ListCards", http.StatusOK, 1013)
}

func TestListCards_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.ListCards(testContext(t), &gomypal.ListCardsRequest{UID: ""})

	assertMissingField(t, err, mock)
}
