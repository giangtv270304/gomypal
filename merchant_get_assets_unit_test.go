package gomypal_test

import (
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestGetMerchantAssets_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/merchant/balance",
		uid:      "",
		body:     `{}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":[{"accountId":100387,"accountType":"MAIN","currency":"USDT","availableAmount":"10","frozenAmount":"1"}]}`,
	})

	resp, err := client.GetMerchantAssets(testContext(t), &gomypal.GetMerchantAssetsRequest{})
	if err != nil {
		t.Fatalf("GetMerchantAssets: %v", err)
	}

	if got, want := len(resp.Data), 1; got != want {
		t.Errorf("len(resp.Data) = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].AccountID, gomypal.ID(100387); got != want {
		t.Errorf("resp.Data[0].AccountID = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].AccountType, gomypal.AccountTypeMain; got != want {
		t.Errorf("resp.Data[0].AccountType = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].Currency, "USDT"; got != want {
		t.Errorf("resp.Data[0].Currency = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].AvailableAmount, gomypal.Decimal("10"); got != want {
		t.Errorf("resp.Data[0].AvailableAmount = %v, want %v", got, want)
	}

	if got, want := resp.Data[0].FrozenAmount, gomypal.Decimal("1"); got != want {
		t.Errorf("resp.Data[0].FrozenAmount = %v, want %v", got, want)
	}
}

func TestGetMerchantAssets_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:     "/merchant/balance",
		uid:      "",
		body:     `{}`,
		status:   http.StatusOK,
		response: `{"code":1301,"msg":"merchant main account not found"}`,
	})

	_, err := client.GetMerchantAssets(testContext(t), &gomypal.GetMerchantAssetsRequest{})

	assertAPIError(t, err, "GetMerchantAssets", http.StatusOK, 1301)
}
