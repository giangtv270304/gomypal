package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestSimulateClear_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/clear",
		uid:      "",
		body:     `{"transactionId":"T1","billAmount":13.07,"transactionAmount":12.5,"transactionCurrency":"EUR"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.SimulateClear(testContext(t), &gomypal.SimulateClearRequest{TransactionID: "T1", BillAmount: "13.07", TransactionAmount: "12.5", TransactionCurrency: "EUR"})
	if err != nil {
		t.Fatalf("SimulateClear: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestSimulateClear_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/clear",
		uid:      "",
		body:     `{"transactionId":"T1","billAmount":13.07,"transactionAmount":12.5,"transactionCurrency":"EUR"}`,
		status:   http.StatusOK,
		response: `{"code":1106,"msg":"card transaction failed"}`,
	})

	_, err := client.SimulateClear(testContext(t), &gomypal.SimulateClearRequest{TransactionID: "T1", BillAmount: "13.07", TransactionAmount: "12.5", TransactionCurrency: "EUR"})

	assertAPIError(t, err, "SimulateClear", http.StatusOK, 1106)
}

func TestSimulateClear_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.SimulateClear(testContext(t), &gomypal.SimulateClearRequest{TransactionID: "", BillAmount: "13.07", TransactionAmount: "12.5", TransactionCurrency: "EUR"})

	assertMissingField(t, err, mock)
}
