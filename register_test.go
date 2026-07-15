package gomypal

import (
	"context"
	"crypto/rsa"
	"net/http"
	"testing"
)

func TestRegister(t *testing.T) {
	client := newTestClient(t, func(pub *rsa.PublicKey) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/user/register" {
				http.NotFound(w, r)
				return
			}

			body := readBody(t, r)
			verifySignature(t, r, body, pub)

			if got := r.Header.Get("uid"); got != "" {
				t.Errorf("uid header = %q, want absent for Register", got)
			}

			writeJSON(t, w, http.StatusOK, map[string]any{"uid": 10001})
		}
	})

	resp, err := client.Register(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UID != 10001 {
		t.Errorf("UID = %d, want 10001", resp.UID)
	}
}
