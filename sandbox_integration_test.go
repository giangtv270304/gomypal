package gomypal_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/giangtv270304/gomypal"
)

const (
	sandboxTimeout      = 30 * time.Second
	sandboxPollInterval = 3 * time.Second
	sandboxPollAttempts = 10
)

func sandboxContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), sandboxTimeout)
	t.Cleanup(cancel)

	return ctx
}

func sandboxCardID(t *testing.T) int64 {
	t.Helper()

	cardID, err := strconv.ParseInt(sandboxEnv(t, envTestCardID), 10, 64)
	if err != nil {
		t.Fatalf("%s is not an integer: %v", envTestCardID, err)
	}

	return cardID
}

type stopTest struct{}

type recordingT struct {
	skipped bool
	failed  bool
}

func (*recordingT) Helper() {}

func (r *recordingT) Skipf(string, ...any) {
	r.skipped = true

	panic(stopTest{})
}

func (r *recordingT) Fatalf(string, ...any) {
	r.failed = true

	panic(stopTest{})
}

func runSandboxHelper(env map[string]string) (outcome *recordingT, client *gomypal.Client) {
	outcome = &recordingT{skipped: false, failed: false}

	defer func() {
		if recovered := recover(); recovered != nil {
			if _, stopped := recovered.(stopTest); !stopped {
				panic(recovered)
			}
		}
	}()

	client = sandboxClientFromEnv(outcome, func(name string) string { return env[name] },
		func(string) ([]byte, error) { return nil, errors.New("no files in this test") })

	return outcome, client
}

func TestSandboxHelper_StopsOnProductionBaseURL(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)

	for _, baseURL := range []string{
		gomypal.ProductionBaseURL, gomypal.ProductionBaseURL + "/", "https://API.cardapi.vip/openapi",
		"https://api.cardapi.vip./openapi", "https://api.cardapi.vip/other", "https://api.cardapi.vip:443",
	} {
		withKeys := map[string]string{envBaseURL: baseURL, envAppID: "app", envPrivateKey: keyBase64}
		withoutKeys := map[string]string{envBaseURL: baseURL}

		for name, env := range map[string]map[string]string{"with keys": withKeys, "without keys": withoutKeys} {
			if outcome, client := runSandboxHelper(env); !outcome.failed || outcome.skipped || client != nil {
				t.Errorf("%s, %s: failed = %v skipped = %v client = %v, want the test to fail and no client", baseURL, name, outcome.failed, outcome.skipped, client)
			}
		}
	}
}

func TestSandboxHelper_SkipsWithoutKeysAndBuildsForASandboxURL(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)

	for name, env := range map[string]map[string]string{
		"nothing set":    {},
		"no base url":    {envAppID: "app", envPrivateKey: keyBase64},
		"no app id":      {envBaseURL: "https://sandbox.example/openapi", envPrivateKey: keyBase64},
		"no private key": {envBaseURL: "https://sandbox.example/openapi", envAppID: "app"},
	} {
		if outcome, client := runSandboxHelper(env); !outcome.skipped || outcome.failed || client != nil {
			t.Errorf("%s: skipped = %v failed = %v, want a skip", name, outcome.skipped, outcome.failed)
		}
	}

	outcome, client := runSandboxHelper(map[string]string{envBaseURL: "https://sandbox.example/openapi", envAppID: "app", envPrivateKey: keyBase64})
	if outcome.skipped || outcome.failed || client == nil {
		t.Errorf("skipped = %v failed = %v client = %v, want a client", outcome.skipped, outcome.failed, client)
	}

	outcome, _ = runSandboxHelper(map[string]string{envBaseURL: "https://sandbox.example/openapi", envAppID: "app", envPrivateKeyFile: "key.txt"})
	if !outcome.failed {
		t.Error("key file that cannot be read: want the test to fail")
	}
}

func TestSandbox_RegisterThenGetCustomerUIDThenKYCStatus(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)
	ctx := sandboxContext(t)
	email := fmt.Sprintf("sdk-sandbox-test+%d@example.com", time.Now().UnixMilli())

	registered, err := client.RegisterCustomer(ctx, &gomypal.RegisterCustomerRequest{Email: email})
	if err != nil {
		t.Fatalf("RegisterCustomer: %v", err)
	}

	if registered.Data.UID == 0 {
		t.Fatal("uid is 0")
	}

	fetched, err := client.GetCustomerUID(ctx, &gomypal.GetCustomerUIDRequest{Email: email})
	if err != nil {
		t.Fatalf("GetCustomerUID: %v", err)
	}

	if fetched.Data.UID != registered.Data.UID {
		t.Errorf("uid = %d, want %d", fetched.Data.UID, registered.Data.UID)
	}

	status, err := client.GetKYCStatus(ctx, &gomypal.GetKYCStatusRequest{UID: strconv.FormatInt(registered.Data.UID, 10)})
	if err != nil {
		t.Fatalf("GetKYCStatus: %v", err)
	}

	if status.Data.Status != gomypal.KYCStatusUncommitted {
		t.Errorf("status = %q, want %q", status.Data.Status, gomypal.KYCStatusUncommitted)
	}

	_, err = client.RegisterCustomer(ctx, &gomypal.RegisterCustomerRequest{Email: email})
	if !gomypal.IsConflict(err) {
		t.Errorf("second RegisterCustomer: error = %v, want code 1011", err)
	}
}

func TestSandbox_ListKYCCountries(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)
	ctx := sandboxContext(t)

	all, err := client.ListKYCCountries(ctx, nil)
	if err != nil {
		t.Fatalf("ListKYCCountries: %v", err)
	}

	if len(all.Data) == 0 {
		t.Error("no countries")
	}

	filtered, err := client.ListKYCCountries(ctx, &gomypal.ListKYCCountriesRequest{Name: "China"})
	if err != nil {
		t.Fatalf("ListKYCCountries(China): %v", err)
	}

	t.Logf("countries: %d, matching China: %d", len(all.Data), len(filtered.Data))
}

func TestSandbox_GetKYCStatus(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.GetKYCStatus(sandboxContext(t), &gomypal.GetKYCStatusRequest{UID: sandboxEnv(t, envTestUID)})
	if err != nil {
		t.Fatalf("GetKYCStatus: %v", err)
	}

	if resp.Data.Status == "" {
		t.Error("status is empty")
	}
}

func TestSandbox_GetKYCStatus_ForeignUID(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	_, err := client.GetKYCStatus(sandboxContext(t), &gomypal.GetKYCStatusRequest{UID: "1"})

	var apiErr *gomypal.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *gomypal.APIError", err)
	}

	t.Logf("foreign uid: http status %d, code %d", apiErr.StatusCode, apiErr.Code)

	if apiErr.Code == 0 {
		t.Errorf("code = 0, want a non-zero code such as 1008 or 1007")
	}
}

func TestSandbox_ListCardProducts(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.ListCardProducts(sandboxContext(t), nil)
	if err != nil {
		t.Fatalf("ListCardProducts: %v", err)
	}

	for _, product := range resp.Data {
		if product.ID == 0 || product.BankCardNature == "" {
			t.Errorf("product = %+v, want id and bankCardNature", product)
		}
	}
}

func TestSandbox_ListCards(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.ListCards(sandboxContext(t), &gomypal.ListCardsRequest{UID: sandboxEnv(t, envTestUID)})
	if err != nil {
		t.Fatalf("ListCards: %v", err)
	}

	t.Logf("cards: %d", len(resp.Data))
}

func TestSandbox_GetCardBalanceAndInfo(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)
	uid := sandboxEnv(t, envTestUID)
	cardID := sandboxCardID(t)
	ctx := sandboxContext(t)

	balance, err := client.GetCardBalance(ctx, &gomypal.GetCardBalanceRequest{UID: uid, UserBankcardID: cardID})
	if err != nil {
		t.Fatalf("GetCardBalance: %v", err)
	}

	if balance.Data.Balance == "" || balance.Data.Currency == "" {
		t.Errorf("balance = %q currency = %q, want both set", balance.Data.Balance, balance.Data.Currency)
	}

	info, err := client.GetCardInfo(ctx, &gomypal.GetCardInfoRequest{UID: uid, UserBankcardID: cardID})
	if err != nil {
		t.Fatalf("GetCardInfo: %v", err)
	}

	t.Logf("card info: infoType %d, status %d", info.Data.InfoType, info.Data.Status)
}

func TestSandbox_ListTransactions(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.ListTransactions(sandboxContext(t), &gomypal.ListTransactionsRequest{
		UID: sandboxEnv(t, envTestUID), UserBankcardID: sandboxCardID(t), PageNum: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}

	t.Logf("transactions: %d", len(resp.Data))
}

func TestSandbox_ListDeliveryRegions(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.ListDeliveryRegions(sandboxContext(t), &gomypal.ListDeliveryRegionsRequest{Local: gomypal.LocaleEnglish})
	if err != nil {
		t.Fatalf("ListDeliveryRegions: %v", err)
	}

	t.Logf("regions: %d", len(resp.Data))
}

func TestSandbox_GetMerchantAssets(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.GetMerchantAssets(sandboxContext(t), nil)
	if err != nil {
		t.Fatalf("GetMerchantAssets: %v", err)
	}

	t.Logf("merchant accounts: %d", len(resp.Data))
}

func TestSandbox_RechargeThenSimulateTrade(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)
	uid := sandboxEnv(t, envTestUID)
	cardID := sandboxCardID(t)

	ctx, cancel := context.WithTimeout(context.Background(), sandboxTimeout+sandboxPollAttempts*sandboxPollInterval)
	defer cancel()

	if _, err := client.GetCardBalance(ctx, &gomypal.GetCardBalanceRequest{UID: uid, UserBankcardID: cardID}); err != nil {
		t.Fatalf("GetCardBalance: %v", err)
	}

	orderID := fmt.Sprintf("sdk-sandbox-test-%d", time.Now().UnixMilli())

	if _, err := client.RechargeCard(ctx, &gomypal.RechargeCardRequest{
		UID: uid, UserBankcardID: cardID, Amount: "1", RequestOrderID: orderID,
	}); err != nil {
		t.Fatalf("RechargeCard: %v", err)
	}

	status := gomypal.OrderStatusPending

	for attempt := 0; attempt < sandboxPollAttempts && status == gomypal.OrderStatusPending; attempt++ {
		select {
		case <-ctx.Done():
			t.Fatalf("waiting for the recharge order: %v", ctx.Err())
		case <-time.After(sandboxPollInterval):
		}

		order, err := client.GetCardBalanceChangeOrder(ctx, &gomypal.GetCardBalanceChangeOrderRequest{
			UID: uid, RequestOrderID: orderID, Type: gomypal.BalanceChangeTypeRecharge,
		})
		if err != nil {
			t.Fatalf("GetCardBalanceChangeOrder: %v", err)
		}

		if order.Data.Status != "" {
			status = order.Data.Status
		}
	}

	t.Logf("recharge order %s: %s", orderID, status)

	if status != gomypal.OrderStatusSuccess {
		t.Skipf("recharge did not reach SUCCESS (%s): the sandbox merchant account may not be funded", status)
	}

	if _, err := client.SimulateTrade(ctx, &gomypal.SimulateTradeRequest{
		UID: uid, BankCardID: cardID, TransactionAmount: "0.5", TransactionCurrency: "USD", BillAmount: "0.5",
	}); err != nil {
		t.Fatalf("SimulateTrade: %v", err)
	}

	transactions, err := client.ListTransactions(ctx, &gomypal.ListTransactionsRequest{
		UID: uid, UserBankcardID: cardID, PageNum: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}

	t.Logf("transactions after the simulated trade: %d", len(transactions.Data))
}
