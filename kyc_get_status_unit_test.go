package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestGetKYCStatus_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/kyc/status",
		uid:      testUID,
		body:     `{}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"status":"success","failedReason":""}}`,
	})

	resp, err := client.GetKYCStatus(testContext(t), &gomypal.GetKYCStatusRequest{UID: testUID})
	if err != nil {
		t.Fatalf("GetKYCStatus: %v", err)
	}

	if got, want := resp.Data.Status, gomypal.KYCStatusSuccess; got != want {
		t.Errorf("resp.Data.Status = %v, want %v", got, want)
	}

	if got, want := resp.Data.FailedReason, ""; got != want {
		t.Errorf("resp.Data.FailedReason = %v, want %v", got, want)
	}
}

func TestGetKYCStatus_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/kyc/status",
		uid:      testUID,
		body:     `{}`,
		status:   http.StatusOK,
		response: `{"code":1008,"msg":"the current user not belong you"}`,
	})

	_, err := client.GetKYCStatus(testContext(t), &gomypal.GetKYCStatusRequest{UID: testUID})

	assertAPIError(t, err, "GetKYCStatus", http.StatusOK, 1008)
}

func TestGetKYCStatus_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.GetKYCStatus(testContext(t), &gomypal.GetKYCStatusRequest{UID: ""})

	assertMissingField(t, err, mock)
}
