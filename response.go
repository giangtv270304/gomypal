package gomypal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// Response is the envelope of every successful call. Code is always 0: a non-zero code is returned as *APIError.
// Data stays at its zero value when MyPal sends null or no data; for a response type without fields data is ignored.
type Response[T any] struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type envelopeFiller interface {
	fill(msg string, data json.RawMessage) error
}

func (r *Response[T]) fill(msg string, data json.RawMessage) error {
	r.Code = int64(CodeSuccess)
	r.Msg = msg

	dataType := reflect.TypeFor[T]()
	if dataType.Kind() == reflect.Struct && dataType.NumField() == 0 {
		return nil
	}

	if text := bytes.TrimSpace(data); len(text) == 0 || string(text) == "null" {
		return nil
	}

	return json.Unmarshal(data, &r.Data)
}

// ID is a numeric id that MyPal sends as a JSON number on some pages and as a numeric string on others.
// It decodes from both (null and "" give 0) and encodes as a JSON number.
type ID int64

// String returns the id in decimal, the form used for the uid header and for string-typed id fields.
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// UnmarshalJSON accepts a JSON integer, a numeric string, "" or null.
func (id *ID) UnmarshalJSON(data []byte) error {
	text := bytes.TrimSpace(data)

	if len(text) > 0 && text[0] == '"' {
		var quoted string
		if err := json.Unmarshal(text, &quoted); err != nil {
			return fmt.Errorf("gomypal: decode id: %w", err)
		}

		text = []byte(quoted)
	}

	if len(text) == 0 || string(text) == "null" {
		*id = 0

		return nil
	}

	value, err := strconv.ParseInt(string(text), 10, 64)
	if err != nil {
		return fmt.Errorf("gomypal: decode id %q: %w", text, err)
	}

	*id = ID(value)

	return nil
}
