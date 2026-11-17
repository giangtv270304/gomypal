package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestSetCardPIN_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/setPin",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"pin":"dsOiXZoeUUBgjw5YujQlvNzMtULGKy3vZcoLgiXgfVc="}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.SetCardPIN(testContext(t), &gomypal.SetCardPINRequest{UID: testUID, UserBankcardID: 100168, PIN: "dsOiXZoeUUBgjw5YujQlvNzMtULGKy3vZcoLgiXgfVc="})
	if err != nil {
		t.Fatalf("SetCardPIN: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestSetCardPIN_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/setPin",
		uid:      testUID,
		body:     `{"userBankcardId":100168,"pin":"dsOiXZoeUUBgjw5YujQlvNzMtULGKy3vZcoLgiXgfVc="}`,
		status:   http.StatusOK,
		response: `{"code":1111,"msg":"card operation not supported"}`,
	})

	_, err := client.SetCardPIN(testContext(t), &gomypal.SetCardPINRequest{UID: testUID, UserBankcardID: 100168, PIN: "dsOiXZoeUUBgjw5YujQlvNzMtULGKy3vZcoLgiXgfVc="})

	assertAPIError(t, err, "SetCardPIN", http.StatusOK, 1111)
}

func TestSetCardPIN_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.SetCardPIN(testContext(t), &gomypal.SetCardPINRequest{UID: testUID, UserBankcardID: 100168})

	assertMissingField(t, err, mock)
}
