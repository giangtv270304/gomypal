package gomypal

import (
	"io"
	"net/http"
)

// request is the internal representation of a single API call. Endpoint
// files only need to set Method, Endpoint, UID (when applicable), and
// Params — everything else is populated by Client.buildRequest.
type request struct {
	Method   string
	Endpoint string

	// UID is the signed-in user's ID. Leave "" for endpoints that don't
	// have one yet (Register) — it still gets signed as an empty value,
	// see the "Signature Notes" caution block, but the HTTP header itself
	// is omitted when empty (see Client.sign).
	UID string

	// Params is the request body, or nil for an empty body ({}).
	Params any

	Header  http.Header
	Body    io.Reader
	FullURL string
}
