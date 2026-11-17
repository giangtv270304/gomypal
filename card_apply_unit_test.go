package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestApplyCard_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/apply",
		uid:      testUID,
		body:     `{"productId":100004}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"userBankcardId":100168,"orderNo":"ORD202511280001"}}`,
	})

	resp, err := client.ApplyCard(testContext(t), &gomypal.ApplyCardRequest{UID: testUID, ProductID: 100004})
	if err != nil {
		t.Fatalf("ApplyCard: %v", err)
	}

	if got, want := resp.Data.UserBankcardID, int64(100168); got != want {
		t.Errorf("resp.Data.UserBankcardID = %v, want %v", got, want)
	}

	if got, want := resp.Data.OrderNo, "ORD202511280001"; got != want {
		t.Errorf("resp.Data.OrderNo = %v, want %v", got, want)
	}

	if got, want := resp.Data.CardNo, ""; got != want {
		t.Errorf("resp.Data.CardNo = %v, want %v", got, want)
	}
}

func TestApplyCard_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/apply",
		uid:      testUID,
		body:     `{"productId":100004}`,
		status:   http.StatusOK,
		response: `{"code":1123,"msg":"duplicate card request"}`,
	})

	_, err := client.ApplyCard(testContext(t), &gomypal.ApplyCardRequest{UID: testUID, ProductID: 100004})

	assertAPIError(t, err, "ApplyCard", http.StatusOK, 1123)
}

func TestApplyCard_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.ApplyCard(testContext(t), &gomypal.ApplyCardRequest{UID: testUID, ProductID: 0})

	assertMissingField(t, err, mock)
}
