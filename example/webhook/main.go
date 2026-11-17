// Webhook: verify and parse MyPal webhooks, and answer 3DS requests that need an approve or reject call.
package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/giangtv270304/gomypal"
)

type server struct {
	client     *gomypal.Client
	webhookKey string
	// uidByCard maps a card id to its MyPal uid. A real app reads this from its own database.
	uidByCard map[string]string
}

func main() {
	client, err := gomypal.NewClient(os.Getenv("MYPAL_BASE_URL"), os.Getenv("MYPAL_APP_ID"), os.Getenv("MYPAL_PRIVATE_KEY"))
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	srv := &server{
		client:     client,
		webhookKey: os.Getenv("MYPAL_WEBHOOK_KEY"),
		uidByCard:  map[string]string{os.Getenv("MYPAL_TEST_CARD_ID"): os.Getenv("MYPAL_TEST_UID")},
	}

	http.HandleFunc("/webhooks/mypal", srv.handleWebhook)

	httpServer := &http.Server{Addr: ":8080", ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(httpServer.ListenAndServe())
}

func (s *server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)

		return
	}

	if err := gomypal.VerifyWebhookSignature(s.webhookKey, body, r.Header.Get(gomypal.WebhookSignHeader)); err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)

		return
	}

	event, err := gomypal.ParseWebhookEvent(body)
	if err != nil {
		writeAck(w, gomypal.WebhookAckFailure())

		return
	}

	switch {
	case event.ThreeDS != nil:
		if err := s.answer3DS(r.Context(), event.ThreeDS); err != nil {
			log.Printf("3ds event %s: %v", event.EventID, err)
			writeAck(w, gomypal.WebhookAckFailure())

			return
		}
	case event.CardRechargeResult != nil:
		log.Printf("recharge order %s: %s", event.CardRechargeResult.OrderID, event.CardRechargeResult.Status)
	default:
		log.Printf("event %s: %s", event.EventID, event.EventType)
	}

	writeAck(w, gomypal.WebhookAckSuccess())
}

func (s *server) answer3DS(ctx context.Context, event *gomypal.ThreeDSEvent) error {
	if event.VerificationType != gomypal.VerificationTypeHTTP {
		return nil
	}

	uid := s.uidByCard[event.UserBankcardID.String()]

	if !userApproved(event) {
		_, err := s.client.Reject3DS(ctx, &gomypal.Reject3DSRequest{UID: uid, AuthID: event.AuthID})

		return err
	}

	_, err := s.client.Approve3DS(ctx, &gomypal.Approve3DSRequest{UID: uid, AuthID: event.AuthID})

	return err
}

// userApproved stands in for asking the card holder (push notification, in-app prompt); replace it.
func userApproved(event *gomypal.ThreeDSEvent) bool {
	log.Printf("3ds request on card %s: %s %s at %s", event.UserBankcardID, event.TransactionAmount, event.TransactionCurrency, event.MerchantName)

	return true
}

func writeAck(w http.ResponseWriter, ack gomypal.WebhookAck) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ack)
}
