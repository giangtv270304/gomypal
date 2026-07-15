package gomypal

import "fmt"

// NOTE: not currently wired up — MyPal's docs don't show an error-body
// shape, only success responses. Client.excute returns a plain
// fmt.Errorf-wrapped ErrUnexpectedStatus on non-2xx today. Revisit this
// once a real error response has been observed from the API.

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gomypal: api error %d: %s", e.StatusCode, e.Message)
}
