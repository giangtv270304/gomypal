package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestUpdateCardEmail_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/update/email",
		uid:      testUID,
		body:     `{"userBankcardId":"100168","email":"new@example.com"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{}}`,
	})

	resp, err := client.UpdateCardEmail(testContext(t), &gomypal.UpdateCardEmailRequest{UID: testUID, UserBankcardID: "100168", Email: "new@example.com"})
	if err != nil {
		t.Fatalf("UpdateCardEmail: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestUpdateCardEmail_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/update/email",
		uid:      testUID,
		body:     `{"userBankcardId":"100168","email":"new@example.com"}`,
		status:   http.StatusOK,
		response: `{"code":1111,"msg":"card operation not supported"}`,
	})

	_, err := client.UpdateCardEmail(testContext(t), &gomypal.UpdateCardEmailRequest{UID: testUID, UserBankcardID: "100168", Email: "new@example.com"})

	assertAPIError(t, err, "UpdateCardEmail", http.StatusOK, 1111)
}

func TestUpdateCardEmail_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.UpdateCardEmail(testContext(t), &gomypal.UpdateCardEmailRequest{UID: testUID, UserBankcardID: "100168"})

	assertMissingField(t, err, mock)
}
