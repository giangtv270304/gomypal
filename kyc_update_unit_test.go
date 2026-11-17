package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestUpdateKYC_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/kyc/update",
		uid:      testUID,
		body:     `{"firstName":"San","lastName":"Zhang","idNo":"S1234567A","email":"san@example.com","nationCode":"SGP","certType":2,"idUrl":"https://files.example/front.png","birthday":"1990-11-26","countryCode":"SGP","areaCode":"65","phone":"81234567"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":null}`,
	})

	resp, err := client.UpdateKYC(testContext(t), &gomypal.UpdateKYCRequest{UID: testUID, KYCInfo: gomypal.KYCInfo{
		FirstName: "San", LastName: "Zhang", IDNo: "S1234567A", Email: "san@example.com", NationCode: "SGP",
		CertType: gomypal.CertTypePassport, IDURL: "https://files.example/front.png", Birthday: "1990-11-26",
		CountryCode: "SGP", AreaCode: "65", Phone: "81234567",
	}})
	if err != nil {
		t.Fatalf("UpdateKYC: %v", err)
	}

	if got, want := resp.Code, int64(0); got != want {
		t.Errorf("resp.Code = %v, want %v", got, want)
	}
}

func TestUpdateKYC_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/kyc/update",
		uid:      testUID,
		body:     `{"firstName":"San","lastName":"Zhang","idNo":"S1234567A","email":"san@example.com","nationCode":"SGP","certType":2,"idUrl":"https://files.example/front.png","birthday":"1990-11-26","countryCode":"SGP","areaCode":"65","phone":"81234567"}`,
		status:   http.StatusOK,
		response: `{"code":1401,"msg":"kyc already approved"}`,
	})

	_, err := client.UpdateKYC(testContext(t), &gomypal.UpdateKYCRequest{UID: testUID, KYCInfo: gomypal.KYCInfo{
		FirstName: "San", LastName: "Zhang", IDNo: "S1234567A", Email: "san@example.com", NationCode: "SGP",
		CertType: gomypal.CertTypePassport, IDURL: "https://files.example/front.png", Birthday: "1990-11-26",
		CountryCode: "SGP", AreaCode: "65", Phone: "81234567",
	}})

	assertAPIError(t, err, "UpdateKYC", http.StatusOK, 1401)
}

func TestUpdateKYC_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.UpdateKYC(testContext(t), &gomypal.UpdateKYCRequest{UID: testUID, KYCInfo: gomypal.KYCInfo{FirstName: "San"}})

	assertMissingField(t, err, mock)
}
