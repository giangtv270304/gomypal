package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestUpdateDeliveryAddress_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/delivery/address/update",
		uid:      testUID,
		body:     `{"id":1000001,"countryRegionId":3,"country":"Singapore","city":"Singapore","receiverName":"San Zhang","receiverMobile":"81234567","receiverAddress":"1 Test Road","postCode":"018956"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"id":"1000001"}}`,
	})

	resp, err := client.UpdateDeliveryAddress(testContext(t), &gomypal.UpdateDeliveryAddressRequest{UID: testUID, ID: 1000001, CountryRegionID: 3, Country: "Singapore", City: "Singapore", ReceiverName: "San Zhang", ReceiverMobile: "81234567", ReceiverAddress: "1 Test Road", PostCode: "018956"})
	if err != nil {
		t.Fatalf("UpdateDeliveryAddress: %v", err)
	}

	if got, want := resp.Data.ID, gomypal.ID(1000001); got != want {
		t.Errorf("resp.Data.ID = %v, want %v", got, want)
	}
}

func TestUpdateDeliveryAddress_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/delivery/address/update",
		uid:      testUID,
		body:     `{"id":1000001,"countryRegionId":3,"country":"Singapore","city":"Singapore","receiverName":"San Zhang","receiverMobile":"81234567","receiverAddress":"1 Test Road","postCode":"018956"}`,
		status:   http.StatusOK,
		response: `{"code":1202,"msg":"delivery address not found"}`,
	})

	_, err := client.UpdateDeliveryAddress(testContext(t), &gomypal.UpdateDeliveryAddressRequest{UID: testUID, ID: 1000001, CountryRegionID: 3, Country: "Singapore", City: "Singapore", ReceiverName: "San Zhang", ReceiverMobile: "81234567", ReceiverAddress: "1 Test Road", PostCode: "018956"})

	assertAPIError(t, err, "UpdateDeliveryAddress", http.StatusOK, 1202)
}

func TestUpdateDeliveryAddress_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.UpdateDeliveryAddress(testContext(t), &gomypal.UpdateDeliveryAddressRequest{UID: testUID, ID: 0, CountryRegionID: 3, Country: "Singapore", City: "Singapore", ReceiverName: "San Zhang", ReceiverMobile: "81234567", ReceiverAddress: "1 Test Road", PostCode: "018956"})

	assertMissingField(t, err, mock)
}
