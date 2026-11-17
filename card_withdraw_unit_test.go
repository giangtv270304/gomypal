package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestWithdrawCard_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/withdraw",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"amount":0.1,"requestOrderId":"order-2"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success"}`,
	})

	resp, err := client.WithdrawCard(testContext(t), &gomypal.WithdrawCardRequest{UID: testUID, UserBankcardID: 100168, Amount: "0.1", RequestOrderID: "order-2"})
	if err != nil {
		t.Fatalf("WithdrawCard: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestWithdrawCard_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/withdraw",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"amount":0.1,"requestOrderId":"order-2"}`,
		status:   http.StatusOK,
		response: `{"code":1104,"msg":"The card is not activated; please try again after activation."}`,
	})

	_, err := client.WithdrawCard(testContext(t), &gomypal.WithdrawCardRequest{UID: testUID, UserBankcardID: 100168, Amount: "0.1", RequestOrderID: "order-2"})

	assertAPIError(t, err, "WithdrawCard", http.StatusOK, 1104)
}

func TestWithdrawCard_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.WithdrawCard(testContext(t), &gomypal.WithdrawCardRequest{UID: testUID, UserBankcardID: 0, Amount: "0.1", RequestOrderID: "order-2"})

	assertMissingField(t, err, mock)
}
