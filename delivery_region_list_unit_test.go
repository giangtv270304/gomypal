package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestListDeliveryRegions_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/delivery/region",
		uid:      "",
		body:     `{"local":"en_US"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":[{"id":3,"countryRegion":"Singapore","countryRegionZh":"新加坡","nationalCode":"0213","price":"48.00"}]}`,
	})

	resp, err := client.ListDeliveryRegions(testContext(t), &gomypal.ListDeliveryRegionsRequest{Local: gomypal.LocaleEnglish})
	if err != nil {
		t.Fatalf("ListDeliveryRegions: %v", err)
	}

	if got, want := len(resp.Data), 1; got != want {
		t.Errorf("len(resp.Data) = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].ID, gomypal.ID(3); got != want {
		t.Errorf("resp.Data[0].ID = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].CountryRegion, "Singapore"; got != want {
		t.Errorf("resp.Data[0].CountryRegion = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].NationalCode, "0213"; got != want {
		t.Errorf("resp.Data[0].NationalCode = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].Price, gomypal.Decimal("48.00"); got != want {
		t.Errorf("resp.Data[0].Price = %v, want %v", got, want)
	}
}

func TestListDeliveryRegions_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/delivery/region",
		uid:      "",
		body:     `{"local":"en_US"}`,
		status:   http.StatusOK,
		response: `{"code":1201,"msg":"delivery region not found"}`,
	})

	_, err := client.ListDeliveryRegions(testContext(t), &gomypal.ListDeliveryRegionsRequest{Local: gomypal.LocaleEnglish})

	assertAPIError(t, err, "ListDeliveryRegions", http.StatusOK, 1201)
}

func TestListDeliveryRegions_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.ListDeliveryRegions(testContext(t), &gomypal.ListDeliveryRegionsRequest{Local: ""})

	assertMissingField(t, err, mock)
}
