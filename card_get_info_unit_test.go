package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestGetCardInfo_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/info",
		uid:      testUID,
		body:     `{"userBankcardId":100168}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"infoType":1,"status":3,"cardNumber":"4288520000001234","cvv":"123","expireDate":"2030-08"}}`,
	})

	resp, err := client.GetCardInfo(testContext(t), &gomypal.GetCardInfoRequest{UID: testUID, UserBankcardID: 100168})
	if err != nil {
		t.Fatalf("GetCardInfo: %v", err)
	}

	if got, want := resp.Data.InfoType, gomypal.CardInfoTypePlain; got != want {
		t.Errorf("resp.Data.InfoType = %v, want %v", got, want)
	}

	if got, want := resp.Data.Status, gomypal.CardStatusNormal; got != want {
		t.Errorf("resp.Data.Status = %v, want %v", got, want)
	}

	if got, want := resp.Data.CardNumber, "4288520000001234"; got != want {
		t.Errorf("resp.Data.CardNumber = %v, want %v", got, want)
	}

	if got, want := resp.Data.CVV, "123"; got != want {
		t.Errorf("resp.Data.CVV = %v, want %v", got, want)
	}

	if got, want := resp.Data.ExpireDate, "2030-08"; got != want {
		t.Errorf("resp.Data.ExpireDate = %v, want %v", got, want)
	}

	if got, want := resp.Data.CardPanURL, ""; got != want {
		t.Errorf("resp.Data.CardPanURL = %v, want %v", got, want)
	}
}

func TestGetCardInfo_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/info",
		uid:      testUID,
		body:     `{"userBankcardId":100168}`,
		status:   http.StatusOK,
		response: `{"code":1102,"msg":"card not found"}`,
	})

	_, err := client.GetCardInfo(testContext(t), &gomypal.GetCardInfoRequest{UID: testUID, UserBankcardID: 100168})

	assertAPIError(t, err, "GetCardInfo", http.StatusOK, 1102)
}

func TestGetCardInfo_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.GetCardInfo(testContext(t), &gomypal.GetCardInfoRequest{UID: testUID, UserBankcardID: 0})

	assertMissingField(t, err, mock)
}
