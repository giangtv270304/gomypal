package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestApprove3DS_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/3ds/approve",
		uid:      testUID,
		body:     `{"authId":"auth-1"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.Approve3DS(testContext(t), &gomypal.Approve3DSRequest{UID: testUID, AuthID: "auth-1"})
	if err != nil {
		t.Fatalf("Approve3DS: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestApprove3DS_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/3ds/approve",
		uid:      testUID,
		body:     `{"authId":"auth-1"}`,
		status:   http.StatusOK,
		response: `{"code":1106,"msg":"card transaction failed"}`,
	})

	_, err := client.Approve3DS(testContext(t), &gomypal.Approve3DSRequest{UID: testUID, AuthID: "auth-1"})

	assertAPIError(t, err, "Approve3DS", http.StatusOK, 1106)
}

func TestApprove3DS_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.Approve3DS(testContext(t), &gomypal.Approve3DSRequest{UID: testUID, AuthID: ""})

	assertMissingField(t, err, mock)
}
