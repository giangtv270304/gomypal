package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestRechargeCard_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/recharge",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"amount":100.5,"requestOrderId":"order-1"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.RechargeCard(testContext(t), &gomypal.RechargeCardRequest{UID: testUID, UserBankcardID: 100168, Amount: "100.5", RequestOrderID: "order-1"})
	if err != nil {
		t.Fatalf("RechargeCard: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestRechargeCard_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/recharge",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"amount":100.5,"requestOrderId":"order-1"}`,
		status:   http.StatusOK,
		response: `{"code":1105,"msg":"insufficient balance"}`,
	})

	_, err := client.RechargeCard(testContext(t), &gomypal.RechargeCardRequest{UID: testUID, UserBankcardID: 100168, Amount: "100.5", RequestOrderID: "order-1"})

	assertAPIError(t, err, "RechargeCard", http.StatusOK, 1105)
}

func TestRechargeCard_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.RechargeCard(testContext(t), &gomypal.RechargeCardRequest{UID: testUID, UserBankcardID: 100168, Amount: "100.5"})

	assertMissingField(t, err, mock)
}
