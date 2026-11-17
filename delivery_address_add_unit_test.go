package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestAddDeliveryAddress_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/delivery/address/add",
		uid:      testUID,
		body:     `{"countryRegionId":3,"country":"Singapore","city":"Singapore","receiverName":"San Zhang","receiverMobile":"81234567","receiverAddress":"1 Test Road","postCode":"018956"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"id":1000001}}`,
	})

	resp, err := client.AddDeliveryAddress(testContext(t), &gomypal.AddDeliveryAddressRequest{UID: testUID, CountryRegionID: 3, Country: "Singapore", City: "Singapore", ReceiverName: "San Zhang", ReceiverMobile: "81234567", ReceiverAddress: "1 Test Road", PostCode: "018956"})
	if err != nil {
		t.Fatalf("AddDeliveryAddress: %v", err)
	}

	if got, want := resp.Data.ID, int64(1000001); got != want {
		t.Errorf("resp.Data.ID = %v, want %v", got, want)
	}
}

func TestAddDeliveryAddress_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/delivery/address/add",
		uid:      testUID,
		body:     `{"countryRegionId":3,"country":"Singapore","city":"Singapore","receiverName":"San Zhang","receiverMobile":"81234567","receiverAddress":"1 Test Road","postCode":"018956"}`,
		status:   http.StatusOK,
		response: `{"code":1203,"msg":"delivery address invalid"}`,
	})

	_, err := client.AddDeliveryAddress(testContext(t), &gomypal.AddDeliveryAddressRequest{UID: testUID, CountryRegionID: 3, Country: "Singapore", City: "Singapore", ReceiverName: "San Zhang", ReceiverMobile: "81234567", ReceiverAddress: "1 Test Road", PostCode: "018956"})

	assertAPIError(t, err, "AddDeliveryAddress", http.StatusOK, 1203)
}

func TestAddDeliveryAddress_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.AddDeliveryAddress(testContext(t), &gomypal.AddDeliveryAddressRequest{UID: testUID, CountryRegionID: 0, Country: "Singapore", City: "Singapore", ReceiverName: "San Zhang", ReceiverMobile: "81234567", ReceiverAddress: "1 Test Road", PostCode: "018956"})

	assertMissingField(t, err, mock)
}
