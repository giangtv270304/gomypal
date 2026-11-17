package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestGetCardBalance_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/balance",
		uid:      testUID,
		body:     `{"userBankcardId":100168}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"balance":"1000","currency":"USD","symbol":"USD","cardLimit":"1200"}}`,
	})

	resp, err := client.GetCardBalance(testContext(t), &gomypal.GetCardBalanceRequest{UID: testUID, UserBankcardID: 100168})
	if err != nil {
		t.Fatalf("GetCardBalance: %v", err)
	}

	if got, want := resp.Data.Balance, gomypal.Decimal("1000"); got != want {
		t.Errorf("resp.Data.Balance = %v, want %v", got, want)
	}

	if got, want := resp.Data.Currency, "USD"; got != want {
		t.Errorf("resp.Data.Currency = %v, want %v", got, want)
	}

	if got, want := resp.Data.Symbol, "USD"; got != want {
		t.Errorf("resp.Data.Symbol = %v, want %v", got, want)
	}

	if got, want := resp.Data.CardLimit, gomypal.Decimal("1200"); got != want {
		t.Errorf("resp.Data.CardLimit = %v, want %v", got, want)
	}
}

func TestGetCardBalance_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/balance",
		uid:      testUID,
		body:     `{"userBankcardId":100168}`,
		status:   http.StatusOK,
		response: `{"code":1102,"msg":"card not found"}`,
	})

	_, err := client.GetCardBalance(testContext(t), &gomypal.GetCardBalanceRequest{UID: testUID, UserBankcardID: 100168})

	assertAPIError(t, err, "GetCardBalance", http.StatusOK, 1102)
}

func TestGetCardBalance_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.GetCardBalance(testContext(t), &gomypal.GetCardBalanceRequest{UID: testUID, UserBankcardID: 0})

	assertMissingField(t, err, mock)
}
