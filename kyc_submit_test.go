package gomypal

import (
	"context"
	"crypto/rsa"
	"net/http"
	"testing"
)

func TestSubmitKYC(t *testing.T) {
	const uid = int64(10001)

	client := newTestClient(t, func(pub *rsa.PublicKey) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/user/kyc/apply" {
				http.NotFound(w, r)
				return
			}

			body := readBody(t, r)
			verifySignature(t, r, body, pub)

			if got := r.Header.Get("uid"); got != "10001" {
				t.Errorf("uid header = %q, want %q", got, "10001")
			}

			writeJSON(t, w, http.StatusOK, map[string]any{})
		}
	})

	err := client.SubmitKYC(context.Background(), uid, &SubmitKYCRequest{
		FirstName:   "SAN",
		LastName:    "ZHANG",
		IDNo:        "1234567890",
		Email:       "user@example.com",
		NationCode:  "SGP",
		CertType:    CertTypeIDCard,
		IDURL:       "https://example.com/id-front.jpg",
		Birthday:    "1990-01-01",
		CountryCode: "SGP",
		AreaCode:    "86",
		Phone:       "12345678",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
