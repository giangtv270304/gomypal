package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestCanActivateCard_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/canActive",
		uid:      testUID,
		body:     `{"cardNo":"4288520000001234","expDate":"08/28"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"canActive":true}}`,
	})

	resp, err := client.CanActivateCard(testContext(t), &gomypal.CanActivateCardRequest{UID: testUID, CardNo: "4288520000001234", ExpDate: "08/28"})
	if err != nil {
		t.Fatalf("CanActivateCard: %v", err)
	}

	if got, want := resp.Data.CanActive, true; got != want {
		t.Errorf("resp.Data.CanActive = %v, want %v", got, want)
	}
}

func TestCanActivateCard_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/canActive",
		uid:      testUID,
		body:     `{"cardNo":"4288520000001234","expDate":"08/28"}`,
		status:   http.StatusOK,
		response: `{"code":1102,"msg":"card not found"}`,
	})

	_, err := client.CanActivateCard(testContext(t), &gomypal.CanActivateCardRequest{UID: testUID, CardNo: "4288520000001234", ExpDate: "08/28"})

	assertAPIError(t, err, "CanActivateCard", http.StatusOK, 1102)
}

func TestCanActivateCard_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.CanActivateCard(testContext(t), &gomypal.CanActivateCardRequest{UID: testUID})

	assertMissingField(t, err, mock)
}
