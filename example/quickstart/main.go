// Quickstart: register a customer, read the uid back and check the KYC state.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/giangtv270304/gomypal"
)

func main() {
	client, err := gomypal.NewClient(
		os.Getenv("MYPAL_BASE_URL"),
		os.Getenv("MYPAL_APP_ID"),
		os.Getenv("MYPAL_PRIVATE_KEY"),
	)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	email := fmt.Sprintf("sdk-sandbox-test+%d@example.com", time.Now().UnixMilli())

	registered, err := client.RegisterCustomer(ctx, &gomypal.RegisterCustomerRequest{Email: email})
	if err != nil {
		var apiErr *gomypal.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("register customer: %s (code %d, http %d)", apiErr.Message, apiErr.Code, apiErr.StatusCode)
		}

		log.Fatalf("register customer: %v", err)
	}

	fetched, err := client.GetCustomerUID(ctx, &gomypal.GetCustomerUIDRequest{Email: email})
	if err != nil {
		log.Fatalf("get customer uid: %v", err)
	}

	uid := strconv.FormatInt(fetched.Data.UID, 10)

	status, err := client.GetKYCStatus(ctx, &gomypal.GetKYCStatusRequest{UID: uid})
	if err != nil {
		log.Fatalf("get kyc status: %v", err)
	}

	fmt.Printf("registered uid %d, read back uid %s, kyc status %s\n", registered.Data.UID, uid, status.Data.Status)
}
