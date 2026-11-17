package gomypal_test

import (
	"encoding/json"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestID_DecodesNumberOrString(t *testing.T) {
	t.Parallel()

	testCases := map[string]gomypal.ID{
		`100168`:             100168,
		`"100168"`:           100168,
		`9007199254740993`:   9007199254740993,
		`"9007199254740993"`: 9007199254740993,
		`""`:                 0,
		`null`:               0,
	}

	for input, want := range testCases {
		var got struct {
			ID gomypal.ID `json:"id"`
		}

		if err := json.Unmarshal([]byte(`{"id":`+input+`}`), &got); err != nil || got.ID != want {
			t.Errorf("unmarshal %s = %d, %v; want %d", input, got.ID, err, want)
		}
	}

	for _, input := range []string{`"abc"`, `1.5`, `true`} {
		var id gomypal.ID
		if err := json.Unmarshal([]byte(input), &id); err == nil {
			t.Errorf("unmarshal %s: want an error", input)
		}
	}

	raw, err := json.Marshal(gomypal.ID(100168))
	if err != nil || string(raw) != "100168" || gomypal.ID(100168).String() != "100168" {
		t.Errorf("marshal = %s, %v; want 100168", raw, err)
	}
}

func TestGetCardInfoResponse_ReadsStatusOrCardStatus(t *testing.T) {
	t.Parallel()

	testCases := map[string]gomypal.CardStatus{
		`{"infoType":0,"status":4,"cardPanUrl":"https://pan.example/x"}`:                gomypal.CardStatusFrozen,
		`{"infoType":0,"cardStatus":3,"cardPanUrl":"https://pan.example/x"}`:            gomypal.CardStatusNormal,
		`{"infoType":0,"status":0,"cardStatus":3,"cardPanUrl":"https://pan.example/x"}`: gomypal.CardStatusInitial,
	}

	for input, want := range testCases {
		var got gomypal.GetCardInfoResponse
		if err := json.Unmarshal([]byte(input), &got); err != nil {
			t.Fatalf("unmarshal %s: %v", input, err)
		}

		if got.Status != want || got.InfoType != gomypal.CardInfoTypeURL || got.CardPanURL != "https://pan.example/x" {
			t.Errorf("unmarshal %s = %+v, want status %d", input, got, want)
		}
	}
}
