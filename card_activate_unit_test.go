package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestActivateCard_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/active",
		uid:      testUID,
		body:     `{"productId":100004,"mobilePrefix":"65","mobile":"81234567","countryCode":"SGP","address":"1 Test Road","city":"Singapore","state":"Singapore","postCode":"018956","verifyCode":"ABC123"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"userBankcardId":1000}}`,
	})

	resp, err := client.ActivateCard(testContext(t), &gomypal.ActivateCardRequest{UID: testUID, ProductID: 100004, MobilePrefix: "65", Mobile: "81234567", CountryCode: "SGP", Address: "1 Test Road", City: "Singapore", State: "Singapore", PostCode: "018956", VerifyCode: "ABC123"})
	if err != nil {
		t.Fatalf("ActivateCard: %v", err)
	}

	if got, want := resp.Data.UserBankcardID, gomypal.ID(1000); got != want {
		t.Errorf("resp.Data.UserBankcardID = %v, want %v", got, want)
	}
}

func TestActivateCard_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/active",
		uid:      testUID,
		body:     `{"productId":100004,"mobilePrefix":"65","mobile":"81234567","countryCode":"SGP","address":"1 Test Road","city":"Singapore","state":"Singapore","postCode":"018956","verifyCode":"ABC123"}`,
		status:   http.StatusOK,
		response: `{"code":1108,"msg":"card already bound"}`,
	})

	_, err := client.ActivateCard(testContext(t), &gomypal.ActivateCardRequest{UID: testUID, ProductID: 100004, MobilePrefix: "65", Mobile: "81234567", CountryCode: "SGP", Address: "1 Test Road", City: "Singapore", State: "Singapore", PostCode: "018956", VerifyCode: "ABC123"})

	assertAPIError(t, err, "ActivateCard", http.StatusOK, 1108)
}

func TestActivateCard_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.ActivateCard(testContext(t), &gomypal.ActivateCardRequest{UID: testUID, ProductID: 100004, MobilePrefix: "65", Mobile: "81234567", CountryCode: "SGP", Address: "1 Test Road", City: "Singapore", State: "Singapore", PostCode: "018956"})

	assertMissingField(t, err, mock)
}
