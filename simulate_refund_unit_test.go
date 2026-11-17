package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestSimulateRefund_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/refund",
		uid:      "",
		body:     `{"transactionId":"T1","transactionAmount":12.5,"billAmount":13.07}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.SimulateRefund(testContext(t), &gomypal.SimulateRefundRequest{TransactionID: "T1", TransactionAmount: "12.5", BillAmount: "13.07"})
	if err != nil {
		t.Fatalf("SimulateRefund: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestSimulateRefund_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/refund",
		uid:      "",
		body:     `{"transactionId":"T1","transactionAmount":12.5,"billAmount":13.07}`,
		status:   http.StatusOK,
		response: `{"code":1106,"msg":"card transaction failed"}`,
	})

	_, err := client.SimulateRefund(testContext(t), &gomypal.SimulateRefundRequest{TransactionID: "T1", TransactionAmount: "12.5", BillAmount: "13.07"})

	assertAPIError(t, err, "SimulateRefund", http.StatusOK, 1106)
}

func TestSimulateRefund_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.SimulateRefund(testContext(t), &gomypal.SimulateRefundRequest{TransactionID: "", TransactionAmount: "12.5", BillAmount: "13.07"})

	assertMissingField(t, err, mock)
}
