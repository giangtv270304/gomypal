package gomypal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrInvalidDecimal is returned before the call when a Decimal in a request is empty or not of the form
// -?digits[.digits], and by RechargeCard and WithdrawCard for an amount that is not above 0 or has more than 2 decimals.
var ErrInvalidDecimal = errors.New("gomypal: decimal must look like 100 or 10.5 (no exponent, not empty)")

var decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

// Decimal is an amount, balance, fee or rate as decimal text, for example "100" or "10.5". It never passes through a float.
// It encodes as a raw JSON number with exactly the digits held, and decodes from a JSON number or a JSON string.
// In requests pass amounts without trailing zeros ("10.5", not "10.50"): only that form is proven to verify.
type Decimal string

// String returns the decimal text.
func (d Decimal) String() string {
	return string(d)
}

// MarshalJSON writes the text as a JSON number. It fails with ErrInvalidDecimal for an empty or malformed value.
func (d Decimal) MarshalJSON() ([]byte, error) {
	if !decimalPattern.MatchString(string(d)) {
		return nil, ErrInvalidDecimal
	}

	return []byte(d), nil
}

// UnmarshalJSON keeps the digits of a JSON number or string as sent. null and "" give an empty Decimal.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	text := bytes.TrimSpace(data)

	switch {
	case len(text) == 0 || string(text) == "null":
		*d = ""
	case text[0] == '"':
		var quoted string
		if err := json.Unmarshal(text, &quoted); err != nil {
			return fmt.Errorf("gomypal: decode decimal: %w", err)
		}

		*d = Decimal(quoted)
	case text[0] == '-' || (text[0] >= '0' && text[0] <= '9'):
		*d = Decimal(text)
	default:
		return fmt.Errorf("gomypal: decode decimal: %w", ErrInvalidDecimal)
	}

	return nil
}

func (d Decimal) checkPositiveAmount(op, name string, maxDecimals int) error {
	text := string(d)
	_, decimals, _ := strings.Cut(text, ".")

	switch {
	case !decimalPattern.MatchString(text):
		return fmt.Errorf("gomypal: %s: %s: %w", op, name, ErrInvalidDecimal)
	case strings.HasPrefix(text, "-") || strings.Trim(text, "0.") == "":
		return fmt.Errorf("gomypal: %s: %s must be greater than 0: %w", op, name, ErrInvalidDecimal)
	case len(decimals) > maxDecimals:
		return fmt.Errorf("gomypal: %s: %s has more than %d decimal places: %w", op, name, maxDecimals, ErrInvalidDecimal)
	}

	return nil
}
