package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestCloseCard_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/close",
		uid:      testUID,
		body:     `{"userBankcardId":"100"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.CloseCard(testContext(t), &gomypal.CloseCardRequest{UID: testUID, UserBankcardID: "100"})
	if err != nil {
		t.Fatalf("CloseCard: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestCloseCard_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/close",
		uid:      testUID,
		body:     `{"userBankcardId":"100"}`,
		status:   http.StatusOK,
		response: `{"code":1109,"msg":"card permission denied"}`,
	})

	_, err := client.CloseCard(testContext(t), &gomypal.CloseCardRequest{UID: testUID, UserBankcardID: "100"})

	assertAPIError(t, err, "CloseCard", http.StatusOK, 1109)
}

func TestCloseCard_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.CloseCard(testContext(t), &gomypal.CloseCardRequest{UID: testUID, UserBankcardID: ""})

	assertMissingField(t, err, mock)
}
