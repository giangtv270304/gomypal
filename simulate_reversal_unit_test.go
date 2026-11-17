package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestSimulateReversal_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/reversal",
		uid:      "",
		body:     `{"transactionId":"T1","transactionAmount":12.5,"billAmount":13.07}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.SimulateReversal(testContext(t), &gomypal.SimulateReversalRequest{TransactionID: "T1", TransactionAmount: "12.5", BillAmount: "13.07"})
	if err != nil {
		t.Fatalf("SimulateReversal: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestSimulateReversal_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/reversal",
		uid:      "",
		body:     `{"transactionId":"T1","transactionAmount":12.5,"billAmount":13.07}`,
		status:   http.StatusOK,
		response: `{"code":1106,"msg":"card transaction failed"}`,
	})

	_, err := client.SimulateReversal(testContext(t), &gomypal.SimulateReversalRequest{TransactionID: "T1", TransactionAmount: "12.5", BillAmount: "13.07"})

	assertAPIError(t, err, "SimulateReversal", http.StatusOK, 1106)
}

func TestSimulateReversal_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.SimulateReversal(testContext(t), &gomypal.SimulateReversalRequest{TransactionID: "T1", TransactionAmount: "12.5"})

	assertMissingField(t, err, mock)
}
