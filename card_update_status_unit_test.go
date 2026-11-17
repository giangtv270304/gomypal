package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestUpdateCardStatus_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/update/status",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"enable":false}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.UpdateCardStatus(testContext(t), &gomypal.UpdateCardStatusRequest{UID: testUID, UserBankcardID: 100168, Enable: false})
	if err != nil {
		t.Fatalf("UpdateCardStatus: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestUpdateCardStatus_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/update/status",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"enable":false}`,
		status:   http.StatusOK,
		response: `{"code":429,"msg":"Operation too frequent"}`,
	})

	_, err := client.UpdateCardStatus(testContext(t), &gomypal.UpdateCardStatusRequest{UID: testUID, UserBankcardID: 100168, Enable: false})

	assertAPIError(t, err, "UpdateCardStatus", http.StatusOK, 429)
}

func TestUpdateCardStatus_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.UpdateCardStatus(testContext(t), &gomypal.UpdateCardStatusRequest{UID: testUID, UserBankcardID: 0, Enable: true})

	assertMissingField(t, err, mock)
}
