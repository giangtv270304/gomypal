package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestSimulateTrade_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/auth",
		uid:      testUID,
		body:     `{"bankCardId":100168,"transactionAmount":12.5,"transactionCurrency":"EUR","billAmount":13.07}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.SimulateTrade(testContext(t), &gomypal.SimulateTradeRequest{UID: testUID, BankCardID: 100168, TransactionAmount: "12.5", TransactionCurrency: "EUR", BillAmount: "13.07"})
	if err != nil {
		t.Fatalf("SimulateTrade: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestSimulateTrade_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/auth",
		uid:      testUID,
		body:     `{"bankCardId":100168,"transactionAmount":12.5,"transactionCurrency":"EUR","billAmount":13.07}`,
		status:   http.StatusOK,
		response: `{"code":1105,"msg":"insufficient balance"}`,
	})

	_, err := client.SimulateTrade(testContext(t), &gomypal.SimulateTradeRequest{UID: testUID, BankCardID: 100168, TransactionAmount: "12.5", TransactionCurrency: "EUR", BillAmount: "13.07"})

	assertAPIError(t, err, "SimulateTrade", http.StatusOK, 1105)
}

func TestSimulateTrade_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.SimulateTrade(testContext(t), &gomypal.SimulateTradeRequest{BankCardID: 100168, TransactionAmount: "12.5", TransactionCurrency: "EUR"})

	assertMissingField(t, err, mock)
}
