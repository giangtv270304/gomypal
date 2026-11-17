package gomypal_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/giangtv270304/gomypal"
)

const (
	docWebhookKey  = "mHy3oMMoHlifsrbU5hbXQV0i5lDspm7UZosRnM/4u80="
	docWebhookSign = "bYcf/qzPEa8n/uwpAIbUFvRNcgBaQo0TC5H3sjRV7bM="
	docWebhookBody = `{"eventType":"kycStatusChange","eventId":"03bb28da57c14e9ba500f25605ae341a","uid":10000,"auditState":1,"auditRemark":"KYC has passed"}`

	transactionCreatedBody = `{"eventType":"transactionCreated","eventId":"evt-2","userBankcardId":100168,"cardNo":"428852******1234",` +
		`"createAt":1764252493756,"currency":"USD","transaction":{"localCurrency":"USD","localCurrencyAmt":"13.07",` +
		`"merchantName":"mypal","transStatus":2,"respCodeDesc":"approved","transType":170,"balance":"986.93","feeAmount":"0.10",` +
		`"feeCurrency":"USD","transactionId":"T1","transCurrencyAmt":"12.50","transCurrency":"EUR","recordNo":"R1",` +
		`"merchantCategoryCode":"5411"}}`
	transactionCreatedSorted = `cardNo=428852******1234&createAt=1764252493756&currency=USD&eventId=evt-2&eventType=transactionCreated&` +
		`transaction={balance:986.93,feeAmount:0.10,feeCurrency:USD,localCurrency:USD,localCurrencyAmt:13.07,merchantCategoryCode:5411,` +
		`merchantName:mypal,recordNo:R1,respCodeDesc:approved,transCurrency:EUR,transCurrencyAmt:12.50,transStatus:2,transType:170,` +
		`transactionId:T1}&userBankcardId=100168`
)

func hmacOf(key, sortedString string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(sortedString))

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhook(t *testing.T) {
	t.Parallel()

	pretty := "{\n  \"eventType\": \"kycStatusChange\",\n  \"eventId\": \"03bb28da57c14e9ba500f25605ae341a\",\n" +
		"  \"uid\": 10000,\n  \"auditState\": 1,\n  \"auditRemark\": \"KYC has passed\"\n}"

	if got := hmacOf(docWebhookKey, "auditRemark=KYC has passed&auditState=1&eventId=03bb28da57c14e9ba500f25605ae341a&eventType=kycStatusChange&uid=10000"); got != docWebhookSign {
		t.Fatalf("doc vector does not reproduce: %s", got)
	}

	accepted := map[string]string{"doc example": docWebhookBody, "doc example pretty-printed": pretty}
	for name, body := range accepted {
		if err := gomypal.VerifyWebhookSignature(docWebhookKey, []byte(body), docWebhookSign); err != nil {
			t.Errorf("%s: error = %v, want nil", name, err)
		}
	}

	changedBody := []byte(docWebhookBody)
	changedBody[len(changedBody)-3] ^= 1

	changedSign := []byte(docWebhookSign)
	changedSign[5] ^= 1

	rejected := []struct {
		name string
		key  string
		body []byte
		sign string
	}{
		{"one byte of the body changed", docWebhookKey, changedBody, docWebhookSign},
		{"one byte of the signature changed", docWebhookKey, []byte(docWebhookBody), string(changedSign)},
		{"uid changed", docWebhookKey, []byte(`{"eventType":"kycStatusChange","eventId":"03bb28da57c14e9ba500f25605ae341a","uid":10001,"auditState":1,"auditRemark":"KYC has passed"}`), docWebhookSign},
		{"wrong key", "another-key", []byte(docWebhookBody), docWebhookSign},
		{"base64-decoded key is not the key", string(mustDecode(t, docWebhookKey)), []byte(docWebhookBody), docWebhookSign},
		{"empty signature", docWebhookKey, []byte(docWebhookBody), ""},
		{"body is not an object", docWebhookKey, []byte(`[1]`), docWebhookSign},
		{"body is not json", docWebhookKey, []byte(`<html>`), docWebhookSign},
	}

	for _, testCase := range rejected {
		err := gomypal.VerifyWebhookSignature(testCase.key, testCase.body, testCase.sign)
		if !errors.Is(err, gomypal.ErrInvalidWebhookSignature) {
			t.Errorf("%s: error = %v, want ErrInvalidWebhookSignature", testCase.name, err)
		}
	}

	if err := gomypal.VerifyWebhookSignature("", []byte(docWebhookBody), docWebhookSign); !errors.Is(err, gomypal.ErrEmptyWebhookKey) {
		t.Errorf("empty key: error = %v, want ErrEmptyWebhookKey", err)
	}
}

func mustDecode(t *testing.T, text string) []byte {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	return raw
}

func TestVerifyWebhook_NestedObject(t *testing.T) {
	t.Parallel()

	sign := hmacOf(docWebhookKey, transactionCreatedSorted)

	if err := gomypal.VerifyWebhookSignature(docWebhookKey, []byte(transactionCreatedBody), sign); err != nil {
		t.Errorf("error = %v, want nil", err)
	}

	withSignField := `{"sign":"ignored",` + transactionCreatedBody[1:]
	if err := gomypal.VerifyWebhookSignature(docWebhookKey, []byte(withSignField), sign); err != nil {
		t.Errorf("body with a sign field: error = %v, want nil", err)
	}
}

func TestVerifyWebhook_EmptyValuesKeptOrDropped(t *testing.T) {
	t.Parallel()

	body := []byte(`{"eventType":"cardStatusChange","eventId":"evt-3","userBankcardId":100168,"cardNo":"4288","status":"cardFreeze",` +
		`"reason":"","refundAmount":null,"extra":{"note":"","keep":1,"gone":null},"empty":{},"list":[1,null,"x"]}`)

	javaSample := "cardNo=4288&empty=&eventId=evt-3&eventType=cardStatusChange&extra={gone:null,keep:1,note:}&list=[1,null,\"x\"]&reason=&refundAmount=null&status=cardFreeze&userBankcardId=100168"
	javaSample = strings.ReplaceAll(javaSample, `"`, "")
	dropNullForm := "cardNo=4288&empty={}&eventId=evt-3&eventType=cardStatusChange&extra={keep:1,note:}&list=[1,x]&reason=&status=cardFreeze&userBankcardId=100168"

	for name, sortedString := range map[string]string{"java sample form": javaSample, "drop-null form": dropNullForm} {
		if err := gomypal.VerifyWebhookSignature(docWebhookKey, body, hmacOf(docWebhookKey, sortedString)); err != nil {
			t.Errorf("%s: error = %v, want nil", name, err)
		}
	}

	others := map[string]string{
		"empty strings dropped too":   "cardNo=4288&eventId=evt-3&eventType=cardStatusChange&extra={keep:1}&list=[1,x]&status=cardFreeze&userBankcardId=100168",
		"null kept in drop-null form": "cardNo=4288&empty={}&eventId=evt-3&eventType=cardStatusChange&extra={gone:null,keep:1,note:}&list=[1,x]&reason=&refundAmount=null&status=cardFreeze&userBankcardId=100168",
	}
	for name, sortedString := range others {
		if err := gomypal.VerifyWebhookSignature(docWebhookKey, body, hmacOf(docWebhookKey, sortedString)); !errors.Is(err, gomypal.ErrInvalidWebhookSignature) {
			t.Errorf("%s: error = %v, want ErrInvalidWebhookSignature", name, err)
		}
	}
}

func rechargeWebhookBody(fields string) string {
	return `{"eventType":"cardRechargeResult","userBankcardId":100168,"cardNo":"4111","rechargeAmount":"10","orderId":"order-1","status":"SUCCESS"` + fields + `}`
}

func TestVerifyWebhook_AmbiguousBodiesAreRejectedEvenWithAMatchingSignature(t *testing.T) {
	t.Parallel()

	original := rechargeWebhookBody(`,"eventId":"E1"`)
	originalSorted := "cardNo=4111&eventId=E1&eventType=cardRechargeResult&orderId=order-1&rechargeAmount=10&status=SUCCESS&userBankcardId=100168"
	sign := hmacOf(docWebhookKey, originalSorted)

	if err := gomypal.VerifyWebhookSignature(docWebhookKey, []byte(original), sign); err != nil {
		t.Fatalf("original body: %v", err)
	}

	tampered := map[string]string{
		"quote inside eventId":  `{"eventType":"cardRechargeResult","userBankcardId":100168,"cardNo":"4111","rechargeAmount":"10","orderId":"order-1","status":"SUCCESS","eventId":"E\"1"}`,
		"pair hidden in cardNo": `{"eventType":"cardRechargeResult","userBankcardId":100168,"cardNo":"4111&eventId=E1","rechargeAmount":"10","orderId":"order-1","status":"SUCCESS"}`,
		"extra key":             `{"eventType":"cardRechargeResult","userBankcardId":100168,"cardNo":"4111","rechargeAmount":"10","orderId":"order-1","status":"SUCCESS","eventId":"E1","x":"y"}`,
	}

	for name, body := range tampered {
		err := gomypal.VerifyWebhookSignature(docWebhookKey, []byte(body), sign)
		if err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	if err := gomypal.VerifyWebhookSignature(docWebhookKey, []byte(tampered["quote inside eventId"]), sign); !errors.Is(err, gomypal.ErrAmbiguousWebhookBody) {
		t.Errorf("quote inside eventId: error = %v, want ErrAmbiguousWebhookBody", err)
	}

	if err := gomypal.VerifyWebhookSignature(docWebhookKey, []byte(tampered["pair hidden in cardNo"]), sign); !errors.Is(err, gomypal.ErrAmbiguousWebhookBody) {
		t.Errorf("pair hidden in cardNo: error = %v, want ErrAmbiguousWebhookBody", err)
	}
}

func TestVerifyWebhook_SignedAmbiguousBodies(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		body   string
		sorted string
	}{
		"keys that differ by case": {
			rechargeWebhookBody(`,"eventId":"E1","EventId":""`),
			"EventId=&cardNo=4111&eventId=E1&eventType=cardRechargeResult&orderId=order-1&rechargeAmount=10&status=SUCCESS&userBankcardId=100168",
		},
		"orderId and orderID": {
			rechargeWebhookBody(`,"eventId":"E1","orderID":""`),
			"cardNo=4111&eventId=E1&eventType=cardRechargeResult&orderID=&orderId=order-1&rechargeAmount=10&status=SUCCESS&userBankcardId=100168",
		},
		"case-differing key in a nested object": {
			`{"eventType":"transactionCreated","eventId":"E1","transaction":{"transactionId":"T1","TransactionId":"T2"}}`,
			"eventId=E1&eventType=transactionCreated&transaction={TransactionId:T2,transactionId:T1}",
		},
		"ampersand in orderId": {
			`{"eventType":"cardRechargeResult","orderId":"o&status=SUCCESS"}`,
			"eventType=cardRechargeResult&orderId=o&status=SUCCESS",
		},
		"equals sign in txHash": {
			`{"eventType":"merchantRecharge","eventId":"E1","txHash":"0xa=b","amount":1}`,
			"amount=1&eventId=E1&eventType=merchantRecharge&txHash=0xa=b",
		},
		"quote in status": {
			`{"eventType":"cardRechargeResult","eventId":"E1","status":"SUC\"CESS"}`,
			"eventId=E1&eventType=cardRechargeResult&status=SUCCESS",
		},
		"pair hidden in a free text value": {
			`{"eventType":"cardStatusChange","eventId":"E1","reason":"x&status=cardClose"}`,
			"eventId=E1&eventType=cardStatusChange&reason=x&status=cardClose",
		},
		"pair hidden in a nested value": {
			`{"eventType":"transactionCreated","eventId":"E1","transaction":{"transactionId":"T1,balance:5"}}`,
			"eventId=E1&eventType=transactionCreated&transaction={transactionId:T1,balance:5}",
		},
	}

	for name, testCase := range testCases {
		sign := hmacOf(docWebhookKey, testCase.sorted)

		if err := gomypal.VerifyWebhookSignature(docWebhookKey, []byte(testCase.body), sign); !errors.Is(err, gomypal.ErrAmbiguousWebhookBody) {
			t.Errorf("%s: Verify error = %v, want ErrAmbiguousWebhookBody", name, err)
		}

		if _, err := gomypal.ParseWebhookEvent([]byte(testCase.body)); !errors.Is(err, gomypal.ErrAmbiguousWebhookBody) {
			t.Errorf("%s: Parse error = %v, want ErrAmbiguousWebhookBody", name, err)
		}
	}
}

func TestParseWebhookEvent_KeysAreCaseSensitive(t *testing.T) {
	t.Parallel()

	event := parseEvent(t, `{"eventType":"cardRechargeResult","eventid":"lower","orderId":"order-1"}`)

	if event.EventID != "" || event.CardRechargeResult == nil || event.CardRechargeResult.OrderID != "order-1" {
		t.Errorf("event = %+v, want only the exact keys read", event)
	}
}

func TestClient_VerifyWebhookSignature(t *testing.T) {
	t.Parallel()

	_, keyBase64 := throwawayKey(t)

	client, err := gomypal.NewClient("https://x.example", testAppID, keyBase64, gomypal.WithWebhookKey(docWebhookKey))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.VerifyWebhookSignature([]byte(docWebhookBody), docWebhookSign); err != nil {
		t.Errorf("error = %v, want nil", err)
	}

	withoutKey, err := gomypal.NewClient("https://x.example", testAppID, keyBase64)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := withoutKey.VerifyWebhookSignature([]byte(docWebhookBody), docWebhookSign); !errors.Is(err, gomypal.ErrEmptyWebhookKey) {
		t.Errorf("no key: error = %v, want ErrEmptyWebhookKey", err)
	}
}

func parseEvent(t *testing.T, body string) *gomypal.WebhookEvent {
	t.Helper()

	event, err := gomypal.ParseWebhookEvent([]byte(body))
	if err != nil {
		t.Fatalf("ParseWebhookEvent(%s): %v", body, err)
	}

	if string(event.Raw) != body {
		t.Errorf("raw = %s, want the body", event.Raw)
	}

	return event
}

func TestParseWebhookEvent_KYCAndUnknown(t *testing.T) {
	t.Parallel()

	event := parseEvent(t, docWebhookBody)
	if event.EventID != "03bb28da57c14e9ba500f25605ae341a" || event.EventType != gomypal.WebhookEventKYCStatusChange {
		t.Errorf("event = %+v", event)
	}

	if kyc := event.KYCStatusChange; kyc == nil || kyc.UID != 10000 || kyc.AuditState != 1 || kyc.AuditRemark != "KYC has passed" {
		t.Errorf("kyc payload = %+v", kyc)
	}

	unknown := parseEvent(t, `{"eventType":"somethingNew","eventId":"evt-9","amount":{"deep":[1,2]}}`)
	if unknown.EventType != "somethingNew" || unknown.EventID != "evt-9" {
		t.Errorf("unknown event = %+v", unknown)
	}

	if unknown.ThreeDS != nil || unknown.CardStatusChange != nil || unknown.CardRechargeResult != nil || unknown.CardWithdrawResult != nil ||
		unknown.TransactionCreated != nil || unknown.MerchantRecharge != nil || unknown.MerchantTradeFee != nil || unknown.KYCStatusChange != nil {
		t.Errorf("unknown event has a payload: %+v", unknown)
	}

	if _, err := gomypal.ParseWebhookEvent([]byte(`not json`)); err == nil {
		t.Error("ParseWebhookEvent(not json): want an error")
	}
}

func TestParseWebhookEvent_CardEvents(t *testing.T) {
	t.Parallel()

	threeDS := parseEvent(t, `{"eventType":"3ds","eventId":"e1","userBankcardId":100168,"verificationType":"http","authId":"auth-1",`+
		`"transactionAmount":"12.50","transactionCurrency":"EUR","merchantName":"mypal","oldUserBankcardId":"77"}`).ThreeDS
	if threeDS == nil || threeDS.UserBankcardID != 100168 || threeDS.VerificationType != gomypal.VerificationTypeHTTP || threeDS.AuthID != "auth-1" ||
		threeDS.TransactionAmount != "12.50" || threeDS.Scene != "" || threeDS.OldUserBankcardID != "77" {
		t.Errorf("3ds = %+v", threeDS)
	}

	otp := parseEvent(t, `{"eventType":"3ds","eventId":"e1","userBankcardId":"100168","verificationType":"otp","otp":"654321","scene":"verification"}`).ThreeDS
	if otp == nil || otp.UserBankcardID != 100168 || otp.OTP != "654321" || otp.Scene != gomypal.ThreeDSSceneVerification {
		t.Errorf("3ds otp = %+v", otp)
	}

	status := parseEvent(t, `{"eventType":"cardStatusChange","eventId":"e2","userBankcardId":"100168","cardNo":"4288","status":"cardClose","refundAmount":"3.20","newField":true}`).CardStatusChange
	if status == nil || status.UserBankcardID != 100168 || status.Status != gomypal.CardStatusEventClose || status.RefundAmount != "3.20" {
		t.Errorf("cardStatusChange = %+v", status)
	}

	recharge := parseEvent(t, `{"eventType":"cardRechargeResult","eventId":"e3","userBankcardId":100168,"cardNo":"4288","rechargeAmount":"100.50","orderId":"order-1","status":"SUCCESS"}`).CardRechargeResult
	if recharge == nil || recharge.RechargeAmount != "100.50" || recharge.OrderID != "order-1" || recharge.Status != gomypal.OrderStatusSuccess {
		t.Errorf("cardRechargeResult = %+v", recharge)
	}

	withdraw := parseEvent(t, `{"eventType":"cardWithdrawResult","eventId":"e4","userBankcardId":100168,"cardNo":"4288","withdrawAmount":"10","fee":"0.10","orderId":"order-2","status":"FAILURE"}`).CardWithdrawResult
	if withdraw == nil || withdraw.WithdrawAmount != "10" || withdraw.Fee != "0.10" || withdraw.OrderID != "order-2" || withdraw.Status != gomypal.OrderStatusFailure {
		t.Errorf("cardWithdrawResult = %+v", withdraw)
	}

	created := parseEvent(t, transactionCreatedBody).TransactionCreated
	if created == nil || created.UserBankcardID != 100168 || created.CreateAt != 1764252493756 || created.Currency != "USD" ||
		created.Transaction.LocalCurrencyAmt != "13.07" || created.Transaction.Balance != "986.93" || created.Transaction.FeeAmount != "0.10" ||
		created.Transaction.TransCurrencyAmt != "12.50" || created.Transaction.TransType != gomypal.TransTypeConsumption ||
		created.Transaction.TransStatus != gomypal.TransStatusSuccess || created.Transaction.MerchantCategoryCode != "5411" {
		t.Errorf("transactionCreated = %+v", created)
	}
}

func TestParseWebhookEvent_MerchantEvents(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"curreny", "currency"} {
		recharge := parseEvent(t, `{"eventType":"merchantRecharge","eventId":"e5","amount":1000.50,"`+key+`":"USDT","txTime":1764252493756,"txHash":"0xabc"}`).MerchantRecharge
		if recharge == nil || recharge.Amount != "1000.50" || recharge.Currency != "USDT" || recharge.TxTime != 1764252493756 || recharge.TxHash != "0xabc" {
			t.Errorf("merchantRecharge with %s = %+v", key, recharge)
		}
	}

	fee := parseEvent(t, `{"eventType":"merchantTradeFee","eventId":"e6","userBankcardId":"100168","orderNo":"N1","amount":"0.30","currency":"USD","createAt":"2025-11-28 18:53:07"}`).MerchantTradeFee
	if fee == nil || fee.UserBankcardID != 100168 || fee.OrderNo != "N1" || fee.Amount != "0.30" || fee.CreateAt != "2025-11-28 18:53:07" {
		t.Errorf("merchantTradeFee = %+v", fee)
	}
}

func TestWebhookAck(t *testing.T) {
	t.Parallel()

	success, _ := json.Marshal(gomypal.WebhookAckSuccess())
	failure, _ := json.Marshal(gomypal.WebhookAckFailure())

	if string(success) != `{"code":0,"msg":"Received successfully"}` || string(failure) != `{"code":1000,"msg":"fail"}` {
		t.Errorf("acks = %s and %s", success, failure)
	}

	if gomypal.WebhookSignHeader != "sign" {
		t.Errorf("WebhookSignHeader = %q, want sign", gomypal.WebhookSignHeader)
	}
}
