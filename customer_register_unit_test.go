package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestRegisterCustomer_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/register",
		uid:      "",
		body:     `{"email":"user@example.com"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"uid":1000001}}`,
	})

	resp, err := client.RegisterCustomer(testContext(t), &gomypal.RegisterCustomerRequest{Email: "user@example.com"})
	if err != nil {
		t.Fatalf("RegisterCustomer: %v", err)
	}

	if got, want := resp.Data.UID, int64(1000001); got != want {
		t.Errorf("resp.Data.UID = %v, want %v", got, want)
	}

	if got, want := resp.Msg, "success"; got != want {
		t.Errorf("resp.Msg = %v, want %v", got, want)
	}
}

func TestRegisterCustomer_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/register",
		uid:      "",
		body:     `{"email":"user@example.com"}`,
		status:   http.StatusOK,
		response: `{"code":1011,"msg":"The email has registered"}`,
	})

	_, err := client.RegisterCustomer(testContext(t), &gomypal.RegisterCustomerRequest{Email: "user@example.com"})

	assertAPIError(t, err, "RegisterCustomer", http.StatusOK, 1011)
}

func TestRegisterCustomer_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.RegisterCustomer(testContext(t), &gomypal.RegisterCustomerRequest{Email: ""})

	assertMissingField(t, err, mock)
}
