package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestGetCardBalanceChangeOrder_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/balance/change/order",
		uid:      testUID,
		body:     `{"requestOrderId":"order-1","type":"RECHARGE"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"requestOrderId":"order-1","status":"PENDING"}}`,
	})

	resp, err := client.GetCardBalanceChangeOrder(testContext(t), &gomypal.GetCardBalanceChangeOrderRequest{UID: testUID, RequestOrderID: "order-1", Type: gomypal.BalanceChangeTypeRecharge})
	if err != nil {
		t.Fatalf("GetCardBalanceChangeOrder: %v", err)
	}

	if got, want := resp.Data.RequestOrderID, "order-1"; got != want {
		t.Errorf("resp.Data.RequestOrderID = %v, want %v", got, want)
	}

	if got, want := resp.Data.Status, gomypal.OrderStatusPending; got != want {
		t.Errorf("resp.Data.Status = %v, want %v", got, want)
	}
}

func TestGetCardBalanceChangeOrder_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/balance/change/order",
		uid:      testUID,
		body:     `{"requestOrderId":"order-1","type":"RECHARGE"}`,
		status:   http.StatusOK,
		response: `{"code":1124,"msg":"order does not exist"}`,
	})

	_, err := client.GetCardBalanceChangeOrder(testContext(t), &gomypal.GetCardBalanceChangeOrderRequest{UID: testUID, RequestOrderID: "order-1", Type: gomypal.BalanceChangeTypeRecharge})

	assertAPIError(t, err, "GetCardBalanceChangeOrder", http.StatusOK, 1124)
}

func TestGetCardBalanceChangeOrder_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.GetCardBalanceChangeOrder(testContext(t), &gomypal.GetCardBalanceChangeOrderRequest{UID: testUID, RequestOrderID: "", Type: gomypal.BalanceChangeTypeRecharge})

	assertMissingField(t, err, mock)
}
