package gomypal_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestDecimal_MarshalWritesTheExactDigitsAsNumber(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"100", "100.5", "0.1", "0.10", "-3.25", "12345678901234567890.123456789012345678"} {
		raw, err := json.Marshal(struct {
			Amount gomypal.Decimal `json:"amount"`
		}{Amount: gomypal.Decimal(text)})
		if err != nil {
			t.Fatalf("marshal %s: %v", text, err)
		}

		if want := `{"amount":` + text + `}`; string(raw) != want {
			t.Errorf("marshal = %s, want %s", raw, want)
		}
	}
}

func TestDecimal_MarshalRejectsMalformedValues(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"", "abc", "1e5", "1.", ".5", "1,5", " 1", "+1", "0x10", `1","x":"y`} {
		if _, err := json.Marshal(gomypal.Decimal(text)); !errors.Is(err, gomypal.ErrInvalidDecimal) {
			t.Errorf("marshal %q: error = %v, want ErrInvalidDecimal", text, err)
		}
	}
}

func TestDecimal_UnmarshalKeepsDigits(t *testing.T) {
	t.Parallel()

	testCases := map[string]gomypal.Decimal{
		`13`:                     "13",
		`0.10`:                   "0.10",
		`9.00000000`:             "9.00000000",
		`"7.00"`:                 "7.00",
		`"0.01000000"`:           "0.01000000",
		`0.1`:                    "0.1",
		`12345678901234567.8901`: "12345678901234567.8901",
		`9007199254740993`:       "9007199254740993",
		`""`:                     "",
		`null`:                   "",
	}

	for input, want := range testCases {
		var got struct {
			Amount gomypal.Decimal `json:"amount"`
		}

		if err := json.Unmarshal([]byte(`{"amount":`+input+`}`), &got); err != nil {
			t.Errorf("unmarshal %s: %v", input, err)

			continue
		}

		if got.Amount != want {
			t.Errorf("unmarshal %s = %q, want %q", input, got.Amount, want)
		}
	}

	var amount gomypal.Decimal
	if err := json.Unmarshal([]byte(`true`), &amount); err == nil {
		t.Error("unmarshal true: want an error")
	}
}

func TestRechargeCard_AmountTextIsSentAndSignedVerbatim(t *testing.T) {
	t.Parallel()

	for _, amount := range []string{"100", "10.5", "0.1", "100.50", "12345678901234567.89"} {
		client := newFakeClient(t, fakeAPI{
			path:     "/bankcard/recharge",
			uid:      testUID,
			body:     `{"userBankcardId":100168,"amount":` + amount + `,"requestOrderId":"order-1"}`,
			status:   http.StatusOK,
			response: successEmpty,
		})

		_, err := client.RechargeCard(testContext(t), &gomypal.RechargeCardRequest{
			UID: testUID, UserBankcardID: 100168, Amount: gomypal.Decimal(amount), RequestOrderID: "order-1",
		})
		if err != nil {
			t.Errorf("RechargeCard %s: %v", amount, err)
		}
	}
}

func TestRechargeCard_MalformedAmountIsRejectedBeforeTheCall(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.RechargeCard(testContext(t), &gomypal.RechargeCardRequest{
		UID: testUID, UserBankcardID: 100168, Amount: "1e3", RequestOrderID: "order-1",
	})
	if !errors.Is(err, gomypal.ErrInvalidDecimal) || mock.calls != 0 {
		t.Errorf("error = %v calls = %d, want ErrInvalidDecimal and no call", err, mock.calls)
	}
}

func TestCardMoney_AmountMustBePositiveWithAtMostTwoDecimals(t *testing.T) {
	t.Parallel()

	for _, amount := range []string{"0", "0.0", "0.00", "-1", "-0.5", "1.234", "0.001", "10.555"} {
		mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
		client := newMockClient(t, mock)

		_, errRecharge := client.RechargeCard(testContext(t), &gomypal.RechargeCardRequest{
			UID: testUID, UserBankcardID: 1, Amount: gomypal.Decimal(amount), RequestOrderID: "o",
		})
		_, errWithdraw := client.WithdrawCard(testContext(t), &gomypal.WithdrawCardRequest{
			UID: testUID, UserBankcardID: 1, Amount: gomypal.Decimal(amount), RequestOrderID: "o",
		})

		if !errors.Is(errRecharge, gomypal.ErrInvalidDecimal) || !errors.Is(errWithdraw, gomypal.ErrInvalidDecimal) || mock.calls != 0 {
			t.Errorf("amount %q: errors = %v and %v, calls = %d, want ErrInvalidDecimal and no call", amount, errRecharge, errWithdraw, mock.calls)
		}
	}

	for _, amount := range []string{"0.01", "1", "10.5", "100.25"} {
		mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
		client := newMockClient(t, mock)

		if _, err := client.RechargeCard(testContext(t), &gomypal.RechargeCardRequest{
			UID: testUID, UserBankcardID: 1, Amount: gomypal.Decimal(amount), RequestOrderID: "o",
		}); err != nil || mock.calls != 1 {
			t.Errorf("amount %q: error = %v calls = %d, want it sent", amount, err, mock.calls)
		}
	}
}
