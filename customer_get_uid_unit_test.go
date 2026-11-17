package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestGetCustomerUID_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/uid",
		uid:      "",
		body:     `{"email":"user@example.com"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"uid":1000001}}`,
	})

	resp, err := client.GetCustomerUID(testContext(t), &gomypal.GetCustomerUIDRequest{Email: "user@example.com"})
	if err != nil {
		t.Fatalf("GetCustomerUID: %v", err)
	}

	if got, want := resp.Data.UID, int64(1000001); got != want {
		t.Errorf("resp.Data.UID = %v, want %v", got, want)
	}
}

func TestGetCustomerUID_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/uid",
		uid:      "",
		body:     `{"email":"user@example.com"}`,
		status:   http.StatusOK,
		response: `{"code":1007,"msg":"the user has not registered"}`,
	})

	_, err := client.GetCustomerUID(testContext(t), &gomypal.GetCustomerUIDRequest{Email: "user@example.com"})

	assertAPIError(t, err, "GetCustomerUID", http.StatusOK, 1007)
}

func TestGetCustomerUID_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.GetCustomerUID(testContext(t), &gomypal.GetCustomerUIDRequest{Email: ""})

	assertMissingField(t, err, mock)
}
