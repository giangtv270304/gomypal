// Recharge: top up a card from the merchant wallet, then poll the order until it settles.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/giangtv270304/gomypal"
)

func main() {
	baseURL := os.Getenv("MYPAL_BASE_URL")
	if strings.HasPrefix(strings.ToLower(baseURL), gomypal.ProductionBaseURL) {
		log.Fatal("MYPAL_BASE_URL points at production; this example moves money, run it on your sandbox")
	}

	client, err := gomypal.NewClient(baseURL, os.Getenv("MYPAL_APP_ID"), os.Getenv("MYPAL_PRIVATE_KEY"))
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	uid := os.Getenv("MYPAL_TEST_UID")

	cardID, err := strconv.ParseInt(os.Getenv("MYPAL_TEST_CARD_ID"), 10, 64)
	if err != nil {
		log.Fatalf("MYPAL_TEST_CARD_ID: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	orderID := fmt.Sprintf("sdk-sandbox-test-%d", time.Now().UnixMilli())

	_, err = client.RechargeCard(ctx, &gomypal.RechargeCardRequest{
		UID: uid, UserBankcardID: cardID, Amount: "1", RequestOrderID: orderID,
	})

	var apiErr *gomypal.APIError

	switch {
	case err == nil:
	case errors.As(err, &apiErr) && apiErr.Code == 1123:
		fmt.Println("order already received, polling it")
	case errors.As(err, &apiErr):
		log.Fatalf("recharge refused: %s (code %d, http %d)", apiErr.Message, apiErr.Code, apiErr.StatusCode)
	default:
		fmt.Printf("recharge outcome unknown (%v), polling order %s\n", err, orderID)
	}

	for {
		order, err := client.GetCardBalanceChangeOrder(ctx, &gomypal.GetCardBalanceChangeOrderRequest{
			UID: uid, RequestOrderID: orderID, Type: gomypal.BalanceChangeTypeRecharge,
		})
		if err != nil {
			log.Fatalf("get order %s: %v", orderID, err)
		}

		if order.Data.Status != gomypal.OrderStatusPending {
			fmt.Printf("order %s: %s\n", orderID, order.Data.Status)

			return
		}

		select {
		case <-ctx.Done():
			log.Fatalf("order %s still pending: %v", orderID, ctx.Err())
		case <-time.After(3 * time.Second):
		}
	}
}
