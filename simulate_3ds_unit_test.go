package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestSimulate3DS_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/3ds",
		uid:      "",
		body:     `{"bankCardId":100168,"transactionAmount":12.5,"transactionCurrency":"USD"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.Simulate3DS(testContext(t), &gomypal.Simulate3DSRequest{BankCardID: 100168, TransactionAmount: "12.5", TransactionCurrency: "USD"})
	if err != nil {
		t.Fatalf("Simulate3DS: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestSimulate3DS_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/simulate/3ds",
		uid:      "",
		body:     `{"bankCardId":100168,"transactionAmount":12.5,"transactionCurrency":"USD"}`,
		status:   http.StatusOK,
		response: `{"code":1102,"msg":"card not found"}`,
	})

	_, err := client.Simulate3DS(testContext(t), &gomypal.Simulate3DSRequest{BankCardID: 100168, TransactionAmount: "12.5", TransactionCurrency: "USD"})

	assertAPIError(t, err, "Simulate3DS", http.StatusOK, 1102)
}

func TestSimulate3DS_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.Simulate3DS(testContext(t), &gomypal.Simulate3DSRequest{BankCardID: 0, TransactionAmount: "12.5", TransactionCurrency: "USD"})

	assertMissingField(t, err, mock)
}
