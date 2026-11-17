package gomypal

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// WebhookSignHeader is the request header that carries the webhook signature.
const WebhookSignHeader = "sign"

var (
	// ErrInvalidWebhookSignature is returned when the sign header does not match the body, or the body is not a JSON object.
	ErrInvalidWebhookSignature = errors.New("gomypal: webhook signature is invalid")
	// ErrAmbiguousWebhookBody is returned for a body whose signature also covers a different body: two keys
	// that differ only by case, a key or an identifier value holding a quote, "&" or "=", or a text value
	// that looks like an extra key=value pair. Treat it like an invalid signature.
	ErrAmbiguousWebhookBody = errors.New("gomypal: webhook body is ambiguous under its signature")

	errWebhookNotObject = errors.New("webhook body is not a JSON object")

	webhookExactTopLevelKeys = map[string]bool{
		"eventId": true, "eventType": true, "orderId": true, "orderNo": true, "status": true, "txHash": true,
		"cardNo": true, "authId": true, "otp": true, "verificationType": true, "scene": true,
		"userBankcardId": true, "oldUserBankcardId": true, "currency": true, "curreny": true,
		"transactionCurrency": true, "transactionAmount": true, "rechargeAmount": true, "withdrawAmount": true,
		"fee": true, "refundAmount": true, "amount": true, "createAt": true,
	}
	webhookFreeTextNestedKeys = map[string]bool{"merchantName": true, "respCodeDesc": true}
)

// WebhookEventType is the eventType of a webhook. Unknown values are kept as sent.
type WebhookEventType string

const (
	WebhookEvent3DS                WebhookEventType = "3ds"
	WebhookEventCardStatusChange   WebhookEventType = "cardStatusChange"
	WebhookEventCardRechargeResult WebhookEventType = "cardRechargeResult"
	WebhookEventCardWithdrawResult WebhookEventType = "cardWithdrawResult"
	WebhookEventTransactionCreated WebhookEventType = "transactionCreated"
	WebhookEventMerchantRecharge   WebhookEventType = "merchantRecharge"
	WebhookEventMerchantTradeFee   WebhookEventType = "merchantTradeFee"
	WebhookEventKYCStatusChange    WebhookEventType = "kycStatusChange"
)

// VerificationType says how a 3DS request is answered.
type VerificationType string

const (
	// VerificationTypeOTP: show ThreeDSEvent.OTP to the user. VerificationTypeHTTP: call Approve3DS or Reject3DS with AuthID.
	VerificationTypeOTP  VerificationType = "otp"
	VerificationTypeHTTP VerificationType = "http"
)

// ThreeDSScene says why a verification code was issued.
type ThreeDSScene string

const (
	ThreeDSSceneVerification ThreeDSScene = "verification"
	ThreeDSScene3DS          ThreeDSScene = "3ds"
)

// CardStatusEvent is the status value of a cardStatusChange webhook.
type CardStatusEvent string

const (
	// The matching CardStatus values are: cardActive 3, cardFreeze 4, cardPreCancelled 6, cardClose 7.
	CardStatusEventFreeze       CardStatusEvent = "cardFreeze"
	CardStatusEventActive       CardStatusEvent = "cardActive"
	CardStatusEventClose        CardStatusEvent = "cardClose"
	CardStatusEventPreCancelled CardStatusEvent = "cardPreCancelled"
)

// WebhookEvent is one decoded webhook. Exactly one payload pointer is set for a known EventType;
// for an unknown EventType all are nil and Raw holds the body.
// A webhook carries no timestamp, so a captured one can be replayed: de-duplicate, and confirm money events
// with GetCardBalanceChangeOrder before crediting.
type WebhookEvent struct {
	// EventID is unique per event. For money events de-duplicate on order id and status (or TxHash) instead.
	EventID   string
	EventType WebhookEventType
	// Raw is the request body as received.
	Raw json.RawMessage

	ThreeDS            *ThreeDSEvent
	CardStatusChange   *CardStatusChangeEvent
	CardRechargeResult *CardRechargeResultEvent
	CardWithdrawResult *CardWithdrawResultEvent
	TransactionCreated *TransactionCreatedEvent
	MerchantRecharge   *MerchantRechargeEvent
	MerchantTradeFee   *MerchantTradeFeeEvent
	KYCStatusChange    *KYCStatusChangeEvent
}

// ThreeDSEvent is the payload of eventType "3ds". OTP is a secret: never log it.
type ThreeDSEvent struct {
	UserBankcardID   ID               `json:"userBankcardId"`
	VerificationType VerificationType `json:"verificationType"`
	// OTP is set for VerificationTypeOTP; AuthID is set for VerificationTypeHTTP.
	OTP                 string  `json:"otp"`
	AuthID              string  `json:"authId"`
	TransactionAmount   Decimal `json:"transactionAmount"`
	TransactionCurrency string  `json:"transactionCurrency"`
	MerchantName        string  `json:"merchantName"`
	// Scene can be empty: MyPal's own example omits it.
	Scene             ThreeDSScene `json:"scene"`
	OldUserBankcardID string       `json:"oldUserBankcardId"`
}

// CardStatusChangeEvent is the payload of eventType "cardStatusChange".
type CardStatusChangeEvent struct {
	UserBankcardID ID              `json:"userBankcardId"`
	CardNo         string          `json:"cardNo"`
	Status         CardStatusEvent `json:"status"`
	// RefundAmount is set only after a card cancellation; Reason is the reason for a freeze.
	RefundAmount      Decimal `json:"refundAmount"`
	Reason            string  `json:"reason"`
	OldUserBankcardID string  `json:"oldUserBankcardId"`
}

// CardRechargeResultEvent is the payload of eventType "cardRechargeResult".
type CardRechargeResultEvent struct {
	UserBankcardID ID      `json:"userBankcardId"`
	CardNo         string  `json:"cardNo"`
	RechargeAmount Decimal `json:"rechargeAmount"`
	// OrderID is the RequestOrderID you sent to RechargeCard.
	OrderID string `json:"orderId"`
	// Status is SUCCESS or FAILURE. It can be empty: MyPal's own example omits it.
	Status            OrderStatus `json:"status"`
	OldUserBankcardID string      `json:"oldUserBankcardId"`
}

// CardWithdrawResultEvent is the payload of eventType "cardWithdrawResult".
type CardWithdrawResultEvent struct {
	UserBankcardID ID      `json:"userBankcardId"`
	CardNo         string  `json:"cardNo"`
	WithdrawAmount Decimal `json:"withdrawAmount"`
	Fee            Decimal `json:"fee"`
	// OrderID is the RequestOrderID you sent to WithdrawCard.
	OrderID           string      `json:"orderId"`
	Status            OrderStatus `json:"status"`
	OldUserBankcardID string      `json:"oldUserBankcardId"`
}

// WebhookTransaction is the transaction object of a transactionCreated webhook.
type WebhookTransaction struct {
	// LocalCurrencyAmt and LocalCurrency are the billed amount and its currency.
	LocalCurrency    string      `json:"localCurrency"`
	LocalCurrencyAmt Decimal     `json:"localCurrencyAmt"`
	MerchantName     string      `json:"merchantName"`
	TransStatus      TransStatus `json:"transStatus"`
	RespCodeDesc     string      `json:"respCodeDesc"`
	TransType        TransType   `json:"transType"`
	// Balance is the card balance after the transaction.
	Balance               Decimal `json:"balance"`
	FeeAmount             Decimal `json:"feeAmount"`
	FeeCurrency           string  `json:"feeCurrency"`
	TransactionID         string  `json:"transactionId"`
	OriginalTransactionID string  `json:"originalTransactionId"`
	// TransCurrencyAmt and TransCurrency are the amount and currency at the merchant.
	TransCurrencyAmt     Decimal `json:"transCurrencyAmt"`
	TransCurrency        string  `json:"transCurrency"`
	RecordNo             string  `json:"recordNo"`
	MerchantCategoryCode string  `json:"merchantCategoryCode"`
}

// TransactionCreatedEvent is the payload of eventType "transactionCreated".
type TransactionCreatedEvent struct {
	UserBankcardID ID     `json:"userBankcardId"`
	CardNo         string `json:"cardNo"`
	// CreateAt is the transaction time as sent; MyPal does not document the unit.
	CreateAt          int64              `json:"createAt"`
	Currency          string             `json:"currency"`
	Transaction       WebhookTransaction `json:"transaction"`
	OldUserBankcardID string             `json:"oldUserBankcardId"`
}

// MerchantRechargeEvent is the payload of eventType "merchantRecharge".
type MerchantRechargeEvent struct {
	// Amount arrives as a JSON number; its digits are kept as sent.
	Amount Decimal `json:"amount"`
	// Currency is read from the key "currency", or from "curreny" (as MyPal documents it) when that is absent.
	Currency string `json:"currency"`
	// TxTime is the transfer time as sent; MyPal does not document the unit.
	TxTime int64  `json:"txTime"`
	TxHash string `json:"txHash"`
}

// MerchantTradeFeeEvent is the payload of eventType "merchantTradeFee".
type MerchantTradeFeeEvent struct {
	UserBankcardID ID      `json:"userBankcardId"`
	OrderNo        string  `json:"orderNo"`
	Amount         Decimal `json:"amount"`
	Currency       string  `json:"currency"`
	// CreateAt is a string here (an integer in TransactionCreatedEvent); MyPal does not document the format.
	CreateAt          string `json:"createAt"`
	OldUserBankcardID string `json:"oldUserBankcardId"`
}

// KYCStatusChangeEvent is the payload of eventType "kycStatusChange". MyPal documents it only by one example.
type KYCStatusChangeEvent struct {
	UID int64 `json:"uid"`
	// AuditState 1 is shown with the remark "KYC has passed"; other values are not documented. Confirm with GetKYCStatus.
	AuditState  int64  `json:"auditState"`
	AuditRemark string `json:"auditRemark"`
}

// WebhookAck is the JSON body MyPal expects in reply to a webhook. Write it with HTTP status 200.
type WebhookAck struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// WebhookAckSuccess is the reply that stops retries: {"code":0,"msg":"Received successfully"}.
func WebhookAckSuccess() WebhookAck {
	return WebhookAck{Code: 0, Msg: "Received successfully"}
}

// WebhookAckFailure is the reply that asks for a retry: {"code":1000,"msg":"fail"}.
// MyPal retries up to 7 times, 1, 2, 4, 8, 15, 30 and 60 minutes after the event.
func WebhookAckFailure() WebhookAck {
	return WebhookAck{Code: 1000, Msg: "fail"}
}

// VerifyWebhookSignature checks the sign header of a webhook against the raw request body.
// webhookKey is the webhook-aesKey text exactly as shown in the merchant dashboard (it is not base64-decoded).
// It returns nil, ErrEmptyWebhookKey, ErrInvalidWebhookSignature (also for an empty signHeader) or,
// when the signature matches but the body can be read in more than one way, ErrAmbiguousWebhookBody.
func VerifyWebhookSignature(webhookKey string, body []byte, signHeader string) error {
	if webhookKey == "" {
		return ErrEmptyWebhookKey
	}

	got := []byte(strings.TrimSpace(signHeader))
	if len(got) == 0 {
		return ErrInvalidWebhookSignature
	}

	fields, err := webhookFields(body)
	if err != nil {
		return ErrInvalidWebhookSignature
	}

	for _, canonical := range []string{webhookSortedString(fields), webhookSortedStringWithoutNull(fields)} {
		mac := hmac.New(sha256.New, []byte(webhookKey))
		mac.Write([]byte(canonical))

		want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		if hmac.Equal(got, []byte(want)) {
			return checkWebhookBody(body, fields)
		}
	}

	return ErrInvalidWebhookSignature
}

// VerifyWebhookSignature is the package VerifyWebhookSignature with the key given to WithWebhookKey.
func (c *Client) VerifyWebhookSignature(body []byte, signHeader string) error {
	return VerifyWebhookSignature(c.webhookKey, body, signHeader)
}

// ParseWebhookEvent decodes a webhook body with exact, case-sensitive key names, the same reading
// VerifyWebhookSignature signs. An unknown eventType is not an error: the payload pointers stay nil and Raw
// holds the body. It returns ErrAmbiguousWebhookBody like VerifyWebhookSignature. It does not check the signature.
func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	fields, err := webhookFields(body)
	if err != nil || len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("gomypal: decode webhook: %w", errWebhookNotObject)
	}

	if err := checkWebhookBody(body, fields); err != nil {
		return nil, err
	}

	event := &WebhookEvent{Raw: bytes.Clone(body)}

	var head struct {
		EventID   string           `json:"eventId"`
		EventType WebhookEventType `json:"eventType"`
	}

	if err := decodeExact(fields, &head); err != nil {
		return nil, fmt.Errorf("gomypal: decode webhook: %w", err)
	}

	event.EventID = head.EventID
	event.EventType = head.EventType

	var payload any

	switch head.EventType {
	case WebhookEvent3DS:
		event.ThreeDS = new(ThreeDSEvent)
		payload = event.ThreeDS
	case WebhookEventCardStatusChange:
		event.CardStatusChange = new(CardStatusChangeEvent)
		payload = event.CardStatusChange
	case WebhookEventCardRechargeResult:
		event.CardRechargeResult = new(CardRechargeResultEvent)
		payload = event.CardRechargeResult
	case WebhookEventCardWithdrawResult:
		event.CardWithdrawResult = new(CardWithdrawResultEvent)
		payload = event.CardWithdrawResult
	case WebhookEventTransactionCreated:
		event.TransactionCreated = new(TransactionCreatedEvent)
		payload = event.TransactionCreated
	case WebhookEventMerchantRecharge:
		event.MerchantRecharge = new(MerchantRechargeEvent)
		payload = event.MerchantRecharge

		if _, present := fields["currency"]; !present {
			fields["currency"] = fields["curreny"]
		}
	case WebhookEventMerchantTradeFee:
		event.MerchantTradeFee = new(MerchantTradeFeeEvent)
		payload = event.MerchantTradeFee
	case WebhookEventKYCStatusChange:
		event.KYCStatusChange = new(KYCStatusChangeEvent)
		payload = event.KYCStatusChange
	default:
		return event, nil
	}

	if err := decodeExact(fields, payload); err != nil {
		return nil, fmt.Errorf("gomypal: decode webhook %s: %w", head.EventType, err)
	}

	return event, nil
}

func decodeExact(fields map[string]json.RawMessage, out any) error {
	target := reflect.ValueOf(out).Elem()

	for index := range target.NumField() {
		name, _, _ := strings.Cut(target.Type().Field(index).Tag.Get("json"), ",")

		raw, present := fields[name]
		if !present || len(raw) == 0 {
			continue
		}

		field := target.Field(index)

		if field.Kind() == reflect.Struct {
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(raw, &nested); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}

			if err := decodeExact(nested, field.Addr().Interface()); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}

			continue
		}

		if err := json.Unmarshal(raw, field.Addr().Interface()); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}

	return nil
}

func webhookFields(body []byte) (map[string]json.RawMessage, error) {
	fields := map[string]json.RawMessage{}

	if len(bytes.TrimSpace(body)) == 0 {
		return fields, nil
	}

	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return nil, errWebhookNotObject
	}

	delete(fields, WebhookSignHeader)

	return fields, nil
}

// MyPal's signed string drops quotes and does not escape "&", "=", "," or ":", so one signature covers several bodies.
func checkWebhookBody(body []byte, fields map[string]json.RawMessage) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if len(bytes.TrimSpace(body)) > 0 && hasAmbiguousKeys(decoder) {
		return ErrAmbiguousWebhookBody
	}

	for key, raw := range fields {
		value := bytes.TrimSpace(raw)

		switch {
		case len(value) == 0:
		case value[0] == '"':
			if ambiguousWebhookText(value, webhookExactTopLevelKeys[key], `"&=`) {
				return ErrAmbiguousWebhookBody
			}
		case value[0] == '{':
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(value, &nested); err != nil {
				return ErrAmbiguousWebhookBody
			}

			for nestedKey, nestedRaw := range nested {
				nestedValue := bytes.TrimSpace(nestedRaw)
				if len(nestedValue) == 0 || nestedValue[0] != '"' {
					continue
				}

				if ambiguousWebhookText(nestedValue, !webhookFreeTextNestedKeys[nestedKey], `"&=,:{}`) {
					return ErrAmbiguousWebhookBody
				}
			}
		}
	}

	return nil
}

func ambiguousWebhookText(raw []byte, exact bool, forbidden string) bool {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return true
	}

	if exact {
		return strings.ContainsAny(text, forbidden) || text == "null"
	}

	return looksLikePair(text, "&", "=") || looksLikePair(text, ",", ":")
}

func looksLikePair(text, pairSeparator, keyValueSeparator string) bool {
	_, rest, found := strings.Cut(text, pairSeparator)

	return found && strings.Contains(rest, keyValueSeparator)
}

func hasAmbiguousKeys(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return true
	}

	switch token {
	case json.Delim('{'):
		var seen []string

		for decoder.More() {
			keyToken, err := decoder.Token()

			key, isString := keyToken.(string)
			if err != nil || !isString || strings.ContainsAny(key, `"&=,:{}`) {
				return true
			}

			for _, earlier := range seen {
				if strings.EqualFold(earlier, key) {
					return true
				}
			}

			seen = append(seen, key)

			if hasAmbiguousKeys(decoder) {
				return true
			}
		}

		_, err := decoder.Token()

		return err != nil
	case json.Delim('['):
		for decoder.More() {
			if hasAmbiguousKeys(decoder) {
				return true
			}
		}

		_, err := decoder.Token()

		return err != nil
	}

	return false
}

func sortedKeys(fields map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

func webhookSortedString(fields map[string]json.RawMessage) string {
	var builder strings.Builder

	for _, key := range sortedKeys(fields) {
		value := bytes.TrimSpace(fields[key])

		builder.WriteString(key + "=")

		var child map[string]json.RawMessage
		if len(value) > 0 && value[0] == '{' && json.Unmarshal(value, &child) == nil {
			builder.WriteString(webhookChildSortedString(child))
		} else {
			builder.WriteString(webhookValueText(value))
		}

		builder.WriteString("&")
	}

	return strings.TrimSuffix(strings.ReplaceAll(builder.String(), `"`, ""), "&")
}

func webhookChildSortedString(child map[string]json.RawMessage) string {
	if len(child) == 0 {
		return ""
	}

	pairs := make([]string, 0, len(child))
	for _, key := range sortedKeys(child) {
		pairs = append(pairs, key+":"+webhookValueText(bytes.TrimSpace(child[key])))
	}

	return "{" + strings.Join(pairs, ",") + "}"
}

func webhookValueText(value []byte) string {
	if len(value) == 0 {
		return ""
	}

	switch value[0] {
	case '"':
		var text string
		if err := json.Unmarshal(value, &text); err == nil {
			return text
		}
	case '{', '[':
		var compact bytes.Buffer
		if err := json.Compact(&compact, value); err == nil {
			return compact.String()
		}
	}

	return string(value)
}

func webhookSortedStringWithoutNull(fields map[string]json.RawMessage) string {
	pairs := make([]string, 0, len(fields))

	for _, key := range sortedKeys(fields) {
		if text := webhookValueTextWithoutNull(fields[key]); text != "" {
			pairs = append(pairs, key+"="+text)
		}
	}

	return strings.ReplaceAll(strings.Join(pairs, "&"), `"`, "")
}

func webhookValueTextWithoutNull(raw json.RawMessage) string {
	value := bytes.TrimSpace(raw)
	if len(value) == 0 || string(value) == "null" {
		return ""
	}

	switch value[0] {
	case '{':
		var child map[string]json.RawMessage
		if err := json.Unmarshal(value, &child); err != nil {
			return ""
		}

		pairs := make([]string, 0, len(child))

		for _, key := range sortedKeys(child) {
			if text := webhookValueTextWithoutNull(child[key]); text != "" {
				pairs = append(pairs, key+":"+text)
			}
		}

		return "{" + strings.Join(pairs, ",") + "}"
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(value, &items); err != nil {
			return ""
		}

		parts := make([]string, 0, len(items))

		for _, item := range items {
			if text := webhookValueTextWithoutNull(item); text != "" {
				parts = append(parts, text)
			}
		}

		return "[" + strings.Join(parts, ",") + "]"
	default:
		return string(value)
	}
}
