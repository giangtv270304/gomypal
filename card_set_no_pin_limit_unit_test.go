package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestSetCardNoPINLimit_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/setNoPinLimit",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"noPinLimit":50}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.SetCardNoPINLimit(testContext(t), &gomypal.SetCardNoPINLimitRequest{UID: testUID, UserBankcardID: 100168, NoPinLimit: "50"})
	if err != nil {
		t.Fatalf("SetCardNoPINLimit: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestSetCardNoPINLimit_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/setNoPinLimit",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"noPinLimit":50}`,
		status:   http.StatusOK,
		response: `{"code":1101,"msg":"card parameter error"}`,
	})

	_, err := client.SetCardNoPINLimit(testContext(t), &gomypal.SetCardNoPINLimitRequest{UID: testUID, UserBankcardID: 100168, NoPinLimit: "50"})

	assertAPIError(t, err, "SetCardNoPINLimit", http.StatusOK, 1101)
}

func TestSetCardNoPINLimit_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.SetCardNoPINLimit(testContext(t), &gomypal.SetCardNoPINLimitRequest{UID: testUID, UserBankcardID: 100168})

	assertMissingField(t, err, mock)
}
