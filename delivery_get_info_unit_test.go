package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestGetDeliveryInfo_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/delivery/info",
		uid:      testUID,
		body:     `{"orderNo":"ORD202511280001"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"deliveryStatus":"DELIVERY","deliveryNo":"SF123456"}}`,
	})

	resp, err := client.GetDeliveryInfo(testContext(t), &gomypal.GetDeliveryInfoRequest{UID: testUID, OrderNo: "ORD202511280001"})
	if err != nil {
		t.Fatalf("GetDeliveryInfo: %v", err)
	}

	if got, want := resp.Data.DeliveryStatus, gomypal.DeliveryStatusDelivery; got != want {
		t.Errorf("resp.Data.DeliveryStatus = %v, want %v", got, want)
	}

	if got, want := resp.Data.DeliveryNo, "SF123456"; got != want {
		t.Errorf("resp.Data.DeliveryNo = %v, want %v", got, want)
	}
}

func TestGetDeliveryInfo_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/delivery/info",
		uid:      testUID,
		body:     `{"orderNo":"ORD202511280001"}`,
		status:   http.StatusOK,
		response: `{"code":1204,"msg":"delivery order not found"}`,
	})

	_, err := client.GetDeliveryInfo(testContext(t), &gomypal.GetDeliveryInfoRequest{UID: testUID, OrderNo: "ORD202511280001"})

	assertAPIError(t, err, "GetDeliveryInfo", http.StatusOK, 1204)
}

func TestGetDeliveryInfo_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.GetDeliveryInfo(testContext(t), &gomypal.GetDeliveryInfoRequest{UID: testUID, OrderNo: ""})

	assertMissingField(t, err, mock)
}
