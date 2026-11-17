package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestListCardProducts_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/merchant/template/list",
		uid:      "",
		body:     `{}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":[{"id":100004,"title":"Virtual USD","bankCardNature":"VIRTUAL","ccy":"USD","applyFee":"7.00","rechargeFee":"0.01000000","bankcardRegion":"HK","cardBin":"428852","activeMinLimit":"20.00000000","rechargeMinLimit":9.00000000}]}`,
	})

	resp, err := client.ListCardProducts(testContext(t), &gomypal.ListCardProductsRequest{})
	if err != nil {
		t.Fatalf("ListCardProducts: %v", err)
	}

	if got, want := len(resp.Data), 1; got != want {
		t.Errorf("len(resp.Data) = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].ID, int64(100004); got != want {
		t.Errorf("resp.Data[0].ID = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].BankCardNature, gomypal.CardTypeVirtual; got != want {
		t.Errorf("resp.Data[0].BankCardNature = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].ApplyFee, gomypal.Decimal("7.00"); got != want {
		t.Errorf("resp.Data[0].ApplyFee = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].RechargeFee, gomypal.Decimal("0.01000000"); got != want {
		t.Errorf("resp.Data[0].RechargeFee = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].ActiveMinLimit, gomypal.Decimal("20.00000000"); got != want {
		t.Errorf("resp.Data[0].ActiveMinLimit = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].RechargeMinLimit, gomypal.Decimal("9.00000000"); got != want {
		t.Errorf("resp.Data[0].RechargeMinLimit = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].CardMode, gomypal.CardMode(""); got != want {
		t.Errorf("resp.Data[0].CardMode = %v, want %v", got, want)
	}
}

func TestListCardProducts_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/bankcard/merchant/template/list",
		uid:      "",
		body:     `{}`,
		status:   http.StatusForbidden,
		response: `{"code":403,"msg":"access denied"}`,
	})

	_, err := client.ListCardProducts(testContext(t), &gomypal.ListCardProductsRequest{})

	assertAPIError(t, err, "ListCardProducts", http.StatusForbidden, 403)
}
