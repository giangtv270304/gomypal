package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestQueryCardPIN_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/queryPin",
		uid:      testUID,
		body:     `{"userBankcardId":"100168"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"pin":"dsOiXZoeUUBgjw5YujQlvNzMtULGKy3vZcoLgiXgfVc="}}`,
	})

	resp, err := client.QueryCardPIN(testContext(t), &gomypal.QueryCardPINRequest{UID: testUID, UserBankcardID: "100168"})
	if err != nil {
		t.Fatalf("QueryCardPIN: %v", err)
	}

	if got, want := resp.Data.PIN, "dsOiXZoeUUBgjw5YujQlvNzMtULGKy3vZcoLgiXgfVc="; got != want {
		t.Errorf("resp.Data.PIN = %v, want %v", got, want)
	}
}

func TestQueryCardPIN_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/queryPin",
		uid:      testUID,
		body:     `{"userBankcardId":"100168"}`,
		status:   http.StatusOK,
		response: `{"code":1111,"msg":"card operation not supported"}`,
	})

	_, err := client.QueryCardPIN(testContext(t), &gomypal.QueryCardPINRequest{UID: testUID, UserBankcardID: "100168"})

	assertAPIError(t, err, "QueryCardPIN", http.StatusOK, 1111)
}

func TestQueryCardPIN_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.QueryCardPIN(testContext(t), &gomypal.QueryCardPINRequest{UID: testUID, UserBankcardID: ""})

	assertMissingField(t, err, mock)
}
