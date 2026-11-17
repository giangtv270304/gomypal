package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestListKYCCountries_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/kyc/country/list",
		uid:      "",
		body:     `{"name":"China"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":[{"name":"中国大陆","enName":"China","value":"CHN","profValue":"CN","code":"156","telephoneCode":"86","needCertificate":true,"extra":1}]}`,
	})

	resp, err := client.ListKYCCountries(testContext(t), &gomypal.ListKYCCountriesRequest{Name: "China"})
	if err != nil {
		t.Fatalf("ListKYCCountries: %v", err)
	}

	if got, want := len(resp.Data), 1; got != want {
		t.Errorf("len(resp.Data) = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].Name, "中国大陆"; got != want {
		t.Errorf("resp.Data[0].Name = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].Value, "CHN"; got != want {
		t.Errorf("resp.Data[0].Value = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].Code, "156"; got != want {
		t.Errorf("resp.Data[0].Code = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].TelephoneCode, "86"; got != want {
		t.Errorf("resp.Data[0].TelephoneCode = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].NeedCertificate, true; got != want {
		t.Errorf("resp.Data[0].NeedCertificate = %v, want %v", got, want)
	}
}

func TestListKYCCountries_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/user/kyc/country/list",
		uid:      "",
		body:     `{"name":"China"}`,
		status:   http.StatusInternalServerError,
		response: `{"code":500,"msg":"system error"}`,
	})

	_, err := client.ListKYCCountries(testContext(t), &gomypal.ListKYCCountriesRequest{Name: "China"})

	assertAPIError(t, err, "ListKYCCountries", http.StatusInternalServerError, 500)
}
