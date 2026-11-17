package gomypal_test

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/giangtv270304/gomypal"
)

type signatureVector struct {
	Name       string `json:"name"`
	AppID      string `json:"appId"`
	Nonce      string `json:"nonce"`
	Timestamp  string `json:"timestamp"`
	UID        string `json:"uid"`
	Body       string `json:"body"`
	PrivateKey string `json:"privateKey"`
	Signature  string `json:"signature"`
}

func loadVector(t *testing.T, name string) signatureVector {
	t.Helper()

	raw, err := os.ReadFile("signer/signature-vectors.json")
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}

	var vectors []signatureVector
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatalf("decode vectors: %v", err)
	}

	for _, vector := range vectors {
		if vector.Name == name {
			return vector
		}
	}

	t.Fatalf("vector %q not found", name)

	return signatureVector{}
}

func rechargeRequest() *gomypal.RechargeCardRequest {
	return &gomypal.RechargeCardRequest{UID: testUID, UserBankcardID: 100168, Amount: "100.5", RequestOrderID: "order-1"}
}

func TestNewClient_RequiredArguments(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)

	testCases := []struct {
		name       string
		baseURL    string
		appID      string
		privateKey string
		want       error
	}{
		{"empty base url", "", "app", keyBase64, gomypal.ErrEmptyBaseURL},
		{"empty app id", "https://x.example", "", keyBase64, gomypal.ErrEmptyAppID},
		{"empty private key", "https://x.example", "app", "", gomypal.ErrEmptyPrivateKey},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := gomypal.NewClient(testCase.baseURL, testCase.appID, testCase.privateKey)
			if !errors.Is(err, testCase.want) {
				t.Errorf("error = %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestNewClient_BaseURLRules(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)

	testCases := map[string]error{
		"https://api.cardapi.vip/openapi":           nil,
		"https://sandbox.example:8443/openapi/":     nil,
		"http://127.0.0.1:8080/openapi":             nil,
		"http://localhost:8080":                     nil,
		"http://[::1]:8080/openapi":                 nil,
		"http://api.cardapi.vip/openapi":            gomypal.ErrInsecureBaseURL,
		"http://127.0.0.1.evil.example/openapi":     gomypal.ErrInsecureBaseURL,
		"http://localhost.evil.example":             gomypal.ErrInsecureBaseURL,
		"ftp://sandbox.example/openapi":             gomypal.ErrInsecureBaseURL,
		"sandbox.example/openapi":                   gomypal.ErrInvalidBaseURL,
		"https://user:secret@sandbox.example":       gomypal.ErrInvalidBaseURL,
		"https://user@sandbox.example":              gomypal.ErrInvalidBaseURL,
		"https://sandbox.example/openapi?token=abc": gomypal.ErrInvalidBaseURL,
		"https://sandbox.example/openapi?":          gomypal.ErrInvalidBaseURL,
		"https://sandbox.example/openapi#frag":      gomypal.ErrInvalidBaseURL,
		"https://":                                  gomypal.ErrInvalidBaseURL,
		"://bad":                                    gomypal.ErrInvalidBaseURL,
	}

	for baseURL, want := range testCases {
		_, err := gomypal.NewClient(baseURL, testAppID, keyBase64)
		if !errors.Is(err, want) {
			t.Errorf("NewClient(%q): error = %v, want %v", baseURL, err, want)
		}

		if err != nil && (strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "token=abc")) {
			t.Errorf("NewClient(%q): error %q echoes the url", baseURL, err)
		}
	}
}

func TestClient_PrintingRedactsAppIDAndKeys(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)

	client, err := gomypal.NewClient("https://sandbox.example/openapi", "app-id-12345", keyBase64,
		gomypal.WithWebhookKey("webhook-key-67890"), gomypal.WithPINKey(docPINKey))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	key, _ := throwawayKey(t)
	secrets := []string{"app-id-12345", "webhook-key-67890", docPINKey, keyBase64[:16], key.D.String(), key.N.String(), key.Primes[0].String()}

	printed := []string{
		fmt.Sprintf("%v", client), fmt.Sprintf("%+v", client), fmt.Sprintf("%#v", client), client.String(), client.GoString(),
		fmt.Sprintf("%v", *client), fmt.Sprintf("%+v", *client), fmt.Sprintf("%#v", *client), fmt.Sprint(client),
	}

	for _, text := range printed {
		if !strings.Contains(text, "https://sandbox.example/openapi") || !strings.Contains(text, "[redacted]") {
			t.Errorf("printed client = %q, want the base url and [redacted]", text)
		}

		for _, secret := range secrets {
			if strings.Contains(text, secret) {
				t.Errorf("printed client %q contains %q", text, secret)
			}
		}
	}
}

func TestExecute_DefaultClientDoesNotFollowRedirects(t *testing.T) {
	t.Parallel()

	var targetCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/elsewhere" {
			targetCalls.Add(1)
			_, _ = w.Write([]byte(successEmpty))

			return
		}

		http.Redirect(w, r, "/elsewhere", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(server.Close)

	_, keyBase64 := throwawayKey(t)

	client, err := gomypal.NewClient(server.URL, testAppID, keyBase64)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.RechargeCard(testContext(t), rechargeRequest())
	if !errors.Is(err, gomypal.ErrInvalidResponse) || resp != nil {
		t.Errorf("resp = %+v err = %v, want ErrInvalidResponse for the redirect response", resp, err)
	}

	if got := targetCalls.Load(); got != 0 {
		t.Errorf("redirect target got %d requests, want 0", got)
	}
}

func TestExecute_ResponseBodyIsCapped(t *testing.T) {
	t.Parallel()

	atLimit := `{"code":0,"msg":"` + strings.Repeat("a", gomypal.MaxResponseBytes-len(`{"code":0,"msg":"","data":{}}`)) + `","data":{}}`

	client := newMockClient(t, &mockHTTPClient{status: http.StatusOK, body: atLimit})
	if _, err := client.RechargeCard(testContext(t), rechargeRequest()); err != nil {
		t.Errorf("body of exactly MaxResponseBytes: %v", err)
	}

	for _, status := range []int{http.StatusOK, http.StatusBadGateway} {
		client := newMockClient(t, &mockHTTPClient{status: status, body: atLimit + " "})

		_, err := client.RechargeCard(testContext(t), rechargeRequest())
		if !errors.Is(err, gomypal.ErrResponseTooLarge) || !errors.Is(err, gomypal.ErrInvalidResponse) {
			t.Errorf("status %d, body one byte over the cap: error = %v, want ErrResponseTooLarge and ErrInvalidResponse", status, err)
		}
	}
}

func TestNewClient_InvalidPrivateKeyIsNotEchoed(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)
	truncated := keyBase64[:len(keyBase64)/2]

	for _, bad := range []string{truncated, "not base64 at all!", base64.StdEncoding.EncodeToString([]byte("not a key"))} {
		_, err := gomypal.NewClient("https://x.example", "app", bad)
		if err == nil {
			t.Fatalf("NewClient accepted an invalid key")
		}

		if strings.Contains(err.Error(), bad) || strings.Contains(err.Error(), bad[:12]) {
			t.Errorf("error %q contains key material", err)
		}
	}
}

func TestNewClient_PrivateKeyFormats(t *testing.T) {
	t.Parallel()

	key, keyBase64 := throwawayKey(t)

	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}

	pkcs1 := x509.MarshalPKCS1PrivateKey(key)

	formats := map[string]string{
		"base64 pkcs8": keyBase64,
		"base64 pkcs1": base64.StdEncoding.EncodeToString(pkcs1),
		"pem pkcs8":    string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})),
		"pem pkcs1":    string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: pkcs1})),
	}

	for name, value := range formats {
		if _, err := gomypal.NewClient("https://x.example", "app", value); err != nil {
			t.Errorf("%s: NewClient: %v", name, err)
		}
	}
}

func TestExecute_FixedSignatureFromVector(t *testing.T) {
	t.Parallel()

	vector := loadVector(t, "derived: number token is signed as sent")
	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}

	client, err := gomypal.NewClient("https://mock.mypal.example/openapi", vector.AppID, vector.PrivateKey,
		gomypal.WithHTTPClient(mock),
		gomypal.WithClock(fixedClock),
		gomypal.WithNonceSource(func() (string, error) { return vector.Nonce, nil }),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.RechargeCard(testContext(t), &gomypal.RechargeCardRequest{
		UID: vector.UID, UserBankcardID: 100168, Amount: "100.5", RequestOrderID: "sdk-order-1",
	})
	if err != nil {
		t.Fatalf("RechargeCard: %v", err)
	}

	if mock.lastBody != vector.Body {
		t.Errorf("body = %s, want %s", mock.lastBody, vector.Body)
	}

	wantHeaders := map[string]string{
		"sign":       vector.Signature,
		"appId":      vector.AppID,
		"nonce":      vector.Nonce,
		"timestamp":  vector.Timestamp,
		"uid":        vector.UID,
		"version":    "v2",
		"User-Agent": "gomypal/0.1.1",
	}
	for name, want := range wantHeaders {
		if got := mock.lastReq.Header.Get(name); got != want {
			t.Errorf("header %s = %q, want %q", name, got, want)
		}
	}
}

func TestExecute_DefaultNonceAndClock(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	before := time.Now().UnixMilli()

	if _, err := client.RechargeCard(testContext(t), rechargeRequest()); err != nil {
		t.Fatalf("RechargeCard: %v", err)
	}

	nonce := mock.lastReq.Header.Get("nonce")
	if len(nonce) != 32 || strings.Trim(nonce, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") != "" {
		t.Errorf("nonce = %q, want 32 characters A-Z0-9", nonce)
	}

	var timestamp int64
	if _, err := fmt.Sscan(mock.lastReq.Header.Get("timestamp"), &timestamp); err != nil || timestamp < before {
		t.Errorf("timestamp = %q, want Unix milliseconds not before %d", mock.lastReq.Header.Get("timestamp"), before)
	}
}

func TestExecute_SpecialCharactersAreSignedAsSent(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path: "/delivery/address/add",
		uid:  testUID,
		body: `{"countryRegionId":3,"country":"Singapore","city":"Singapore","receiverName":"Tom \"TJ\" Lee",` +
			`"receiverMobile":"81234567","receiverAddress":"1 A&B <Road>","postCode":"018956"}`,
		status:   http.StatusOK,
		response: `{"code":0,"msg":"success","data":{"id":1}}`,
	})

	_, err := client.AddDeliveryAddress(testContext(t), &gomypal.AddDeliveryAddressRequest{
		UID: testUID, CountryRegionID: 3, Country: "Singapore", City: "Singapore", ReceiverName: `Tom "TJ" Lee`,
		ReceiverMobile: "81234567", ReceiverAddress: "1 A&B <Road>", PostCode: "018956",
	})
	if err != nil {
		t.Fatalf("AddDeliveryAddress: %v", err)
	}
}

func TestExecute_NonZeroCodeOnHTTP200IsAPIError(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, &mockHTTPClient{status: http.StatusOK, body: `{"code":429,"msg":"Operation too frequent"}`})

	resp, err := client.RechargeCard(testContext(t), rechargeRequest())
	if resp != nil {
		t.Errorf("response = %+v, want nil", resp)
	}

	assertAPIError(t, err, "RechargeCard", http.StatusOK, gomypal.CodeTooManyRequests)

	apiErr := apiErrorOf(t, err)
	if apiErr.Message != "Operation too frequent" || string(apiErr.Body) != `{"code":429,"msg":"Operation too frequent"}` {
		t.Errorf("message = %q body = %s", apiErr.Message, apiErr.Body)
	}

	if !gomypal.IsRateLimited(err) {
		t.Error("IsRateLimited = false, want true")
	}

	want := "gomypal: api error: op=RechargeCard status=200 code=429 message=Operation too frequent"
	if err.Error() != want {
		t.Errorf("error text = %q, want %q", err.Error(), want)
	}
}

func TestExecute_NonJSONErrorBodyIsKept(t *testing.T) {
	t.Parallel()

	html := "<html><body>502 Bad Gateway</body></html>"
	client := newMockClient(t, &mockHTTPClient{status: http.StatusBadGateway, body: html})

	_, err := client.GetMerchantAssets(testContext(t), nil)

	assertAPIError(t, err, "GetMerchantAssets", http.StatusBadGateway, 0)

	apiErr := apiErrorOf(t, err)
	if apiErr.Message != html || string(apiErr.Body) != html {
		t.Errorf("message = %q body = %q, want the raw html", apiErr.Message, apiErr.Body)
	}
}

func TestExecute_LongNonJSONErrorBodyIsCutInMessageOnly(t *testing.T) {
	t.Parallel()

	long := "<html>" + strings.Repeat("x", 5000) + "</html>"
	client := newMockClient(t, &mockHTTPClient{status: http.StatusBadGateway, body: long})

	_, err := client.GetMerchantAssets(testContext(t), nil)

	assertAPIError(t, err, "GetMerchantAssets", http.StatusBadGateway, 0)

	apiErr := apiErrorOf(t, err)
	if apiErr.Message != long[:200] || string(apiErr.Body) != long {
		t.Errorf("message has %d bytes and body %d, want 200 and %d", len(apiErr.Message), len(apiErr.Body), len(long))
	}

	if len(err.Error()) > 400 {
		t.Errorf("error text has %d bytes, want it short", len(err.Error()))
	}
}

func TestExecute_HTTPErrorWithEnvelope(t *testing.T) {
	t.Parallel()

	client := newMockClient(t, &mockHTTPClient{status: http.StatusTooManyRequests, body: `{"code":429,"msg":"too many requests, please try again later"}`})

	_, err := client.GetMerchantAssets(testContext(t), nil)

	assertAPIError(t, err, "GetMerchantAssets", http.StatusTooManyRequests, gomypal.CodeTooManyRequests)
}

func TestExecute_BrokenSuccessBodyIsError(t *testing.T) {
	t.Parallel()

	bodies := map[string]string{
		"truncated json":   `{"code":0,"msg":"success","data":`,
		"html":             "<html>ok</html>",
		"empty":            "",
		"no envelope code": `{"uid":1}`,
		"wrong data type":  `{"code":0,"msg":"success","data":"oops"}`,
		"data is a list":   `{"code":0,"msg":"success","data":[1]}`,
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := newMockClient(t, &mockHTTPClient{status: http.StatusOK, body: body})

			resp, err := client.GetCustomerUID(testContext(t), &gomypal.GetCustomerUIDRequest{Email: "user@example.com"})
			if err == nil || resp != nil {
				t.Fatalf("resp = %+v err = %v, want nil response and an error", resp, err)
			}

			var apiErr *gomypal.APIError
			if errors.As(err, &apiErr) || !errors.Is(err, gomypal.ErrInvalidResponse) {
				t.Errorf("error = %v, want ErrInvalidResponse and not *APIError", err)
			}
		})
	}
}

func TestExecute_NullOrAbsentDataIsEmptyResponse(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`{"code":0,"msg":"success","data":null}`, `{"code":0,"msg":"success"}`} {
		client := newMockClient(t, &mockHTTPClient{status: http.StatusOK, body: body})

		order, err := client.GetCardBalanceChangeOrder(testContext(t), &gomypal.GetCardBalanceChangeOrderRequest{
			UID: testUID, RequestOrderID: "order-1", Type: gomypal.BalanceChangeTypeWithdraw,
		})
		if err != nil || order.Data.Status != "" {
			t.Errorf("body %s: order = %+v err = %v, want empty data and no error", body, order, err)
		}

		cards, err := client.ListCards(testContext(t), &gomypal.ListCardsRequest{UID: testUID})
		if err != nil || len(cards.Data) != 0 {
			t.Errorf("body %s: cards = %+v err = %v, want no cards and no error", body, cards, err)
		}
	}
}

func TestExecute_MoneyCallOutcomes(t *testing.T) {
	t.Parallel()

	ignored := newMockClient(t, &mockHTTPClient{status: http.StatusOK, body: `{"code":0,"msg":"success","data":"ok"}`})
	if resp, err := ignored.RechargeCard(testContext(t), rechargeRequest()); err != nil || resp.Code != 0 || resp.Msg != "success" {
		t.Errorf("code 0 with data \"ok\" for an empty response type: resp = %+v err = %v, want success", resp, err)
	}

	for _, status := range []int{http.StatusMovedPermanently, http.StatusFound, http.StatusNotModified, http.StatusContinue} {
		client := newMockClient(t, &mockHTTPClient{status: status, body: `{"code":0,"msg":"success"}`})

		resp, err := client.RechargeCard(testContext(t), rechargeRequest())
		if !errors.Is(err, gomypal.ErrInvalidResponse) || resp != nil {
			t.Errorf("status %d: resp = %+v err = %v, want ErrInvalidResponse", status, resp, err)
		}

		var apiErr *gomypal.APIError
		if errors.As(err, &apiErr) {
			t.Errorf("status %d: error is an *APIError, want an unknown outcome", status)
		}
	}

	reset := newMockClient(t, &mockHTTPClient{err: syscall.ECONNRESET})

	_, err := reset.WithdrawCard(testContext(t), &gomypal.WithdrawCardRequest{UID: testUID, UserBankcardID: 1, Amount: "1", RequestOrderID: "o"})

	var apiErr *gomypal.APIError
	if !errors.Is(err, syscall.ECONNRESET) || errors.As(err, &apiErr) || errors.Is(err, gomypal.ErrMissingField) || errors.Is(err, gomypal.ErrInvalidDecimal) {
		t.Errorf("connection reset: error = %v, want a wrapped transport error", err)
	}
}

func TestExecute_NetworkErrorIsWrapped(t *testing.T) {
	t.Parallel()

	errConnection := errors.New("connection refused")
	client := newMockClient(t, &mockHTTPClient{err: errConnection})

	_, err := client.GetMerchantAssets(testContext(t), nil)
	if !errors.Is(err, errConnection) {
		t.Errorf("error = %v, want it to wrap the transport error", err)
	}
}

func TestExecute_Timeout(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))

	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	_, keyBase64 := throwawayKey(t)

	client, err := gomypal.NewClient(server.URL, testAppID, keyBase64, gomypal.WithTimeout(50*time.Millisecond))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetMerchantAssets(context.Background(), nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestExecute_NilRequest(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.RechargeCard(testContext(t), nil)
	if !errors.Is(err, gomypal.ErrNilRequest) || mock.calls != 0 {
		t.Errorf("error = %v calls = %d, want ErrNilRequest and no call", err, mock.calls)
	}
}

func TestExecute_EmptyUIDIsRejectedBeforeTheCall(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)
	ctx := testContext(t)

	calls := map[string]func() error{
		"GetKYCStatus": func() error {
			_, err := client.GetKYCStatus(ctx, &gomypal.GetKYCStatusRequest{UID: ""})

			return err
		},
		"RechargeCard": func() error {
			_, err := client.RechargeCard(ctx, &gomypal.RechargeCardRequest{UID: "", UserBankcardID: 1, Amount: "1", RequestOrderID: "o"})

			return err
		},
		"WithdrawCard": func() error {
			_, err := client.WithdrawCard(ctx, &gomypal.WithdrawCardRequest{UID: "", UserBankcardID: 1, Amount: "1", RequestOrderID: "o"})

			return err
		},
		"GetCardInfo": func() error {
			_, err := client.GetCardInfo(ctx, &gomypal.GetCardInfoRequest{UID: "", UserBankcardID: 1})

			return err
		},
		"Approve3DS": func() error {
			_, err := client.Approve3DS(ctx, &gomypal.Approve3DSRequest{UID: "", AuthID: "a"})

			return err
		},
		"CloseCard": func() error {
			_, err := client.CloseCard(ctx, &gomypal.CloseCardRequest{UID: "", UserBankcardID: "1"})

			return err
		},
		"ListTransactions": func() error {
			_, err := client.ListTransactions(ctx, &gomypal.ListTransactionsRequest{UID: "", UserBankcardID: 1})

			return err
		},
		"GetDeliveryInfo": func() error {
			_, err := client.GetDeliveryInfo(ctx, &gomypal.GetDeliveryInfoRequest{UID: "", OrderNo: "o"})

			return err
		},
	}

	for name, run := range calls {
		if err := run(); !errors.Is(err, gomypal.ErrMissingField) || !strings.Contains(err.Error(), "uid") {
			t.Errorf("%s: error = %v, want ErrMissingField naming uid", name, err)
		}
	}

	if mock.calls != 0 {
		t.Errorf("http calls = %d, want 0", mock.calls)
	}
}

func TestSimulate_RefusedOnProductionBaseURL(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)
	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}

	client, err := gomypal.NewClient(gomypal.ProductionBaseURL+"/", testAppID, keyBase64, gomypal.WithHTTPClient(mock))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := testContext(t)

	for _, variant := range []string{"https://API.CardAPI.vip/openapi", "https://api.cardapi.vip:443/openapi/v2", "https://api.cardapi.vip", "https://api.cardapi.vip./openapi", "https://API.CARDAPI.VIP.:8443"} {
		other, err := gomypal.NewClient(variant, testAppID, keyBase64, gomypal.WithHTTPClient(mock))
		if err != nil {
			t.Fatalf("NewClient(%s): %v", variant, err)
		}

		_, err = other.SimulateRefund(ctx, &gomypal.SimulateRefundRequest{TransactionID: "T1", TransactionAmount: "1", BillAmount: "1"})
		if !errors.Is(err, gomypal.ErrSandboxOnly) {
			t.Errorf("%s: error = %v, want ErrSandboxOnly", variant, err)
		}
	}

	_, err3DS := client.Simulate3DS(ctx, &gomypal.Simulate3DSRequest{BankCardID: 1, TransactionAmount: "1", TransactionCurrency: "USD"})
	_, errTrade := client.SimulateTrade(ctx, &gomypal.SimulateTradeRequest{BankCardID: 1, TransactionAmount: "1", TransactionCurrency: "USD", BillAmount: "1"})
	_, errRefund := client.SimulateRefund(ctx, &gomypal.SimulateRefundRequest{TransactionID: "T1", TransactionAmount: "1", BillAmount: "1"})
	_, errClear := client.SimulateClear(ctx, &gomypal.SimulateClearRequest{TransactionID: "T1", BillAmount: "1", TransactionAmount: "1", TransactionCurrency: "USD"})
	_, errReversal := client.SimulateReversal(ctx, &gomypal.SimulateReversalRequest{TransactionID: "T1", TransactionAmount: "1", BillAmount: "1"})

	for name, err := range map[string]error{"3ds": err3DS, "trade": errTrade, "refund": errRefund, "clear": errClear, "reversal": errReversal} {
		if !errors.Is(err, gomypal.ErrSandboxOnly) {
			t.Errorf("%s: error = %v, want ErrSandboxOnly", name, err)
		}
	}

	if mock.calls != 0 {
		t.Errorf("http calls = %d, want 0", mock.calls)
	}

	if _, err := client.GetKYCStatus(ctx, &gomypal.GetKYCStatusRequest{UID: testUID}); err != nil || mock.calls != 1 {
		t.Errorf("GetKYCStatus on production: err = %v calls = %d, want it to be sent", err, mock.calls)
	}
}

type recordingLogger struct {
	lines []string
}

func (l *recordingLogger) Debug(msg string, args ...any) {
	l.lines = append(l.lines, fmt.Sprint(append([]any{msg}, args...)...))
}

func TestLogger_NeverLogsHeadersBodyOrKeys(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)
	logger := &recordingLogger{lines: nil}
	mock := &mockHTTPClient{
		status: http.StatusOK,
		body:   `{"code":0,"msg":"success","data":{"infoType":1,"status":3,"cardNumber":"4288520000001234","cvv":"987","expireDate":"2030-08"}}`,
	}

	client, err := gomypal.NewClient("https://mock.mypal.example/openapi", testAppID, keyBase64,
		gomypal.WithHTTPClient(mock), gomypal.WithLogger(logger), gomypal.WithNonceSource(fixedNonce), gomypal.WithClock(fixedClock))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.GetCardInfo(testContext(t), &gomypal.GetCardInfoRequest{UID: testUID, UserBankcardID: 100168}); err != nil {
		t.Fatalf("GetCardInfo: %v", err)
	}

	if _, err := client.RechargeCard(testContext(t), &gomypal.RechargeCardRequest{
		UID: testUID, UserBankcardID: 100168, Amount: "77.25", RequestOrderID: "order-secret-1",
	}); err != nil {
		t.Fatalf("RechargeCard: %v", err)
	}

	logged := strings.Join(logger.lines, "\n")
	if len(logger.lines) != 2 || !strings.Contains(logged, "/bankcard/info") || !strings.Contains(logged, "/bankcard/recharge") || !strings.Contains(logged, "POST") || !strings.Contains(logged, "200") {
		t.Fatalf("log = %q, want two lines with method, path and status", logged)
	}

	forbidden := []string{
		"mock.mypal.example", "https://", "?",
		mock.lastReq.Header.Get("sign"), testNonce, testTimestamp, testAppID, keyBase64, keyBase64[:24],
		"4288520000001234", "987", "77.25", "order-secret-1", "userBankcardId",
	}
	for _, secret := range forbidden {
		if strings.Contains(logged, secret) {
			t.Errorf("log contains %q", secret)
		}
	}
}
