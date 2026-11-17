package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestReject3DS_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/3ds/reject",
		uid:      testUID,
		body:     `{"authId":"auth-1"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.Reject3DS(testContext(t), &gomypal.Reject3DSRequest{UID: testUID, AuthID: "auth-1"})
	if err != nil {
		t.Fatalf("Reject3DS: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestReject3DS_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/3ds/reject",
		uid:      testUID,
		body:     `{"authId":"auth-1"}`,
		status:   http.StatusOK,
		response: `{"code":1106,"msg":"card transaction failed"}`,
	})

	_, err := client.Reject3DS(testContext(t), &gomypal.Reject3DSRequest{UID: testUID, AuthID: "auth-1"})

	assertAPIError(t, err, "Reject3DS", http.StatusOK, 1106)
}

func TestReject3DS_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.Reject3DS(testContext(t), &gomypal.Reject3DSRequest{UID: testUID, AuthID: ""})

	assertMissingField(t, err, mock)
}
