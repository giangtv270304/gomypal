// Command sandbox is a manual smoke-test tool for exercising gomypal
// against a real MyPal sandbox environment. It is NOT a go test — it
// makes real network calls and (with -submit-kyc) creates a real user
// record, so it's driven by explicit flags/env vars and run by hand.
//
// Usage:
//
//	export MYPAL_BASE_URL="https://sandbox.mypal.example"   # real sandbox host from MyPal
//	export MYPAL_APP_ID="your-sandbox-app-id"
//	export MYPAL_PRIVATE_KEY_FILE="/path/to/sandbox-private-key.pem"  # or MYPAL_PRIVATE_KEY="<bare base64 PKCS8>"
//
//	go run ./examples/sandbox -email "you+test1@example.com"
//	go run ./examples/sandbox -email "you+test2@example.com" -submit-kyc -check-status
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomypal"
)

func main() {
	email := flag.String("email", "", "email to register (required; use a fresh address each run — sandboxes usually reject re-registering the same email)")
	submitKYC := flag.Bool("submit-kyc", false, "also submit a dummy SubmitKYC payload for the newly registered uid")
	checkStatus := flag.Bool("check-status", false, "also call CheckKYCStatus for the uid (register/submit first, or pass -uid to check an existing one)")
	uidFlag := flag.Int64("uid", 0, "skip Register and use this uid directly (for -submit-kyc/-check-status against an already-registered user)")
	flag.Parse()

	baseURL := requireEnv("MYPAL_BASE_URL")
	appID := requireEnv("MYPAL_APP_ID")
	privateKey := loadPrivateKey()

	client, err := gomypal.NewClient(baseURL, appID, privateKey, gomypal.WithDebug(true))
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uid := *uidFlag
	if uid == 0 {
		if *email == "" {
			log.Fatal("-email is required unless -uid is set (pick a fresh address — most sandboxes reject re-registering the same email)")
		}
		resp, err := client.Register(ctx, *email)
		if err != nil {
			log.Fatalf("Register: %v", err)
		}
		uid = resp.UID
		fmt.Printf("Register OK: uid=%d\n", uid)
	}

	if *submitKYC {
		err := client.SubmitKYC(ctx, uid, &gomypal.SubmitKYCRequest{
			FirstName:   "SAN",
			LastName:    "ZHANG",
			IDNo:        "1234567890",
			Email:       *email,
			NationCode:  "SGP",
			CertType:    gomypal.CertTypeIDCard,
			IDURL:       "https://example.com/id-front.jpg",
			Birthday:    "1990-01-01",
			CountryCode: "SGP",
			AreaCode:    "86",
			Phone:       "12345678",
		})
		if err != nil {
			log.Fatalf("SubmitKYC: %v", err)
		}
		fmt.Println("SubmitKYC OK")
	}

	if *checkStatus {
		resp, err := client.CheckKYCStatus(ctx, uid)
		if err != nil {
			log.Fatalf("CheckKYCStatus: %v", err)
		}
		fmt.Printf("CheckKYCStatus OK: status=%s failedReason=%q\n", resp.Status, resp.FailedReason)
	}
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("environment variable %s is required", name)
	}
	return v
}

// loadPrivateKey reads either MYPAL_PRIVATE_KEY (the raw key string
// directly) or MYPAL_PRIVATE_KEY_FILE (a path to a file containing it,
// PEM or bare base64 PKCS8 — gomypal.NewClient accepts both forms).
func loadPrivateKey() string {
	if v := os.Getenv("MYPAL_PRIVATE_KEY"); v != "" {
		return v
	}
	path := os.Getenv("MYPAL_PRIVATE_KEY_FILE")
	if path == "" {
		log.Fatal("set MYPAL_PRIVATE_KEY or MYPAL_PRIVATE_KEY_FILE")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read MYPAL_PRIVATE_KEY_FILE: %v", err)
	}
	return string(data)
}
