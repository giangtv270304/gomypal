package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestListTransactions_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/trade/details",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"pageSize":10,"pageNum":1}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":[{"id":1001,"localCurrencyAmt":13,"localCurrency":"USD","recordNo":"R202511280001","occurTime":"2025-11-28 18:53:07","respCodeDesc":"approved","feeAmount":0,"feeCurrency":"USD","merchantName":"mypal","transType":202,"transStatus":2,"transactionId":"T1","transCurrencyAmt":"","transCurrency":"","merchantCategoryCode":"5411","accountNum":"A1","failureReason":""}]}`,
	})

	resp, err := client.ListTransactions(testContext(t), &gomypal.ListTransactionsRequest{UID: testUID, UserBankcardID: 100168, PageSize: 10, PageNum: 1})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}

	if got, want := len(resp.Data), 1; got != want {
		t.Errorf("len(resp.Data) = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].ID, int64(1001); got != want {
		t.Errorf("resp.Data[0].ID = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].LocalCurrencyAmt, gomypal.Decimal("13"); got != want {
		t.Errorf("resp.Data[0].LocalCurrencyAmt = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].FeeAmount, gomypal.Decimal("0"); got != want {
		t.Errorf("resp.Data[0].FeeAmount = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].TransCurrencyAmt, gomypal.Decimal(""); got != want {
		t.Errorf("resp.Data[0].TransCurrencyAmt = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].OccurTime, "2025-11-28 18:53:07"; got != want {
		t.Errorf("resp.Data[0].OccurTime = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].TransType, gomypal.TransType(202); got != want {
		t.Errorf("resp.Data[0].TransType = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].TransStatus, gomypal.TransStatusSuccess; got != want {
		t.Errorf("resp.Data[0].TransStatus = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].AccountNum, "A1"; got != want {
		t.Errorf("resp.Data[0].AccountNum = %v, want %v", got, want)
	}
}

func TestListTransactions_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/trade/details",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"pageSize":10,"pageNum":1}`,
		status:   http.StatusOK,
		response: `{"code":1102,"msg":"card not found"}`,
	})

	_, err := client.ListTransactions(testContext(t), &gomypal.ListTransactionsRequest{UID: testUID, UserBankcardID: 100168, PageSize: 10, PageNum: 1})

	assertAPIError(t, err, "ListTransactions", http.StatusOK, 1102)
}

func TestListTransactions_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.ListTransactions(testContext(t), &gomypal.ListTransactionsRequest{UID: testUID, UserBankcardID: 0})

	assertMissingField(t, err, mock)
}
