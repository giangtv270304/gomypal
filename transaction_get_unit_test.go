package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestGetTransaction_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/trade/info",
		uid:      testUID,
		body:     `{"id":1001}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"id":1001,"userBankcardId":100168,"localCurrencyAmt":13.07,"localCurrency":"USD","recordNo":"R202511280001","occurTime":"2025-11-28 18:53:07","respCodeDesc":"approved","feeAmount":0.10,"feeCurrency":"USD","merchantName":"mypal","transType":170,"transStatus":2,"transactionId":"T1","originalTransactionId":"T0"}}`,
	})

	resp, err := client.GetTransaction(testContext(t), &gomypal.GetTransactionRequest{UID: testUID, ID: 1001})
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}

	if got, want := resp.Data.ID, int64(1001); got != want {
		t.Errorf("resp.Data.ID = %v, want %v", got, want)
	}

	if got, want := resp.Data.UserBankcardID, int64(100168); got != want {
		t.Errorf("resp.Data.UserBankcardID = %v, want %v", got, want)
	}

	if got, want := resp.Data.LocalCurrencyAmt, gomypal.Decimal("13.07"); got != want {
		t.Errorf("resp.Data.LocalCurrencyAmt = %v, want %v", got, want)
	}

	if got, want := resp.Data.FeeAmount, gomypal.Decimal("0.10"); got != want {
		t.Errorf("resp.Data.FeeAmount = %v, want %v", got, want)
	}

	if got, want := resp.Data.TransType, gomypal.TransTypeConsumption; got != want {
		t.Errorf("resp.Data.TransType = %v, want %v", got, want)
	}

	if got, want := resp.Data.TransactionID, "T1"; got != want {
		t.Errorf("resp.Data.TransactionID = %v, want %v", got, want)
	}

	if got, want := resp.Data.OriginalTransactionID, "T0"; got != want {
		t.Errorf("resp.Data.OriginalTransactionID = %v, want %v", got, want)
	}
}

func TestGetTransaction_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/trade/info",
		uid:      testUID,
		body:     `{"id":1001}`,
		status:   http.StatusOK,
		response: `{"code":1000,"msg":"business processing failed"}`,
	})

	_, err := client.GetTransaction(testContext(t), &gomypal.GetTransactionRequest{UID: testUID, ID: 1001})

	assertAPIError(t, err, "GetTransaction", http.StatusOK, 1000)
}

func TestGetTransaction_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.GetTransaction(testContext(t), &gomypal.GetTransactionRequest{UID: testUID, ID: 0})

	assertMissingField(t, err, mock)
}
