package gomypal

import (
	"context"
	"crypto/rsa"
	"net/http"
	"testing"
)

func TestCheckKYCStatus(t *testing.T) {
	const uid = int64(10001)

	client := newTestClient(t, func(pub *rsa.PublicKey) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/user/kyc/status" {
				http.NotFound(w, r)
				return
			}

			body := readBody(t, r)
			verifySignature(t, r, body, pub)

			if got := r.Header.Get("uid"); got != "10001" {
				t.Errorf("uid header = %q, want %q", got, "10001")
			}

			writeJSON(t, w, http.StatusOK, map[string]any{"status": "success", "failedReason": ""})
		}
	})

	resp, err := client.CheckKYCStatus(context.Background(), uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != KYCStatusSuccess {
		t.Errorf("Status = %q, want %q", resp.Status, KYCStatusSuccess)
	}
}
