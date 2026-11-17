# gomypal

Go client for the MyPal card API (customers, KYC, cards, transactions, delivery, merchant wallet, webhooks).

## Install

```bash
go get github.com/giangtv270304/gomypal
```

Requires Go 1.23+. No dependencies outside the standard library.

## First call

Get the base URL, `appId` and RSA private key of your MyPal sandbox merchant, then:

```go
client, err := gomypal.NewClient(baseURL, appID, privateKey)
if err != nil {
	log.Fatal(err)
}

resp, err := client.RegisterCustomer(ctx, &gomypal.RegisterCustomerRequest{Email: "user@example.com"})
if err != nil {
	log.Fatal(err)
}

fmt.Println(resp.Data.UID)
```

A full program is in `example/quickstart`.

## Authentication

Every request is signed with your RSA private key ("Signature Description v2": RSA-SHA1 over the signing headers and the JSON body). The client does it for you: it encodes the body once, signs those bytes and sends those bytes.

The private key, the webhook key and the PIN key are for servers only. Never ship them in a browser or a mobile app, and never log them. `NewClient` takes the key as base64 of PKCS#8 DER (the form MyPal issues); base64 PKCS#1 and PEM are accepted too. A key that cannot be parsed is an error at construction, and the error never contains the key.

| Environment | Base URL |
|---|---|
| Production | `gomypal.ProductionBaseURL` (`https://api.cardapi.vip/openapi`) |
| Sandbox | MyPal publishes no sandbox URL. There is no sandbox constant: pass the URL MyPal gave you |

The base URL must be `https` (plain `http` is accepted only for `localhost`, `127.0.0.1` and `::1`) and must not carry user info, a query or a fragment.

Most calls act for one user: set `UID` on the request to the uid returned by `RegisterCustomer` (as decimal text). An empty `UID` is rejected before the call.

## Flows

Runnable programs, configured from `.env.example`: `example/quickstart` (register and KYC status), `example/recharge` (top up a card and poll the order; refuses the production URL), `example/webhook` (verify, parse, answer 3DS).

### Register a user and pass KYC

1. `RegisterCustomer` returns the `uid`. `GetCustomerUID` finds it again by email.
2. `ListKYCCountries` tells which countries need extra documents (`NeedCertificate`).
3. `UploadKYCFile` for each document; put the returned `FileURL` into `KYCInfo`.
4. `SubmitKYC` (or `UpdateKYC`), then poll `GetKYCStatus` or wait for the `kycStatusChange` webhook.

### Issue a card and top it up

1. `ListCardProducts`, then `ApplyCard` with the product id. A physical card also needs `AddDeliveryAddress` first.
2. `RechargeCard` with your own unique `RequestOrderID`. The response only says the order was accepted.
3. Wait for the `cardRechargeResult` webhook, or poll `GetCardBalanceChangeOrder` until `SUCCESS` or `FAILURE`.

```go
_, err := client.RechargeCard(ctx, &gomypal.RechargeCardRequest{
	UID: uid, UserBankcardID: cardID, Amount: "10.5", RequestOrderID: orderID,
})
```

`Amount` must be above 0 with at most 2 decimals; anything else fails before the call.

What an error on `RechargeCard` or `WithdrawCard` means:

| Result | Meaning |
|---|---|
| `ErrMissingField`, `ErrInvalidDecimal`, `ErrNilRequest`, `ErrSandboxOnly` | nothing was sent |
| `*APIError` with code `1104`, `1105`, `429` | MyPal did not process the order; seen in production. Fix the cause (activate the card, fund it, slow down) and send again |
| `*APIError` with code `1001`, `1002`, `1003`, `1004`, `1005`, `1006`, `1008`, `1014`, `403` | rejected by validation or authentication, which by their meaning in MyPal's code table happens before the order is processed |
| `*APIError` with code `1123` | MyPal already holds this `RequestOrderID`: the first send was received; query the order |
| `*APIError` with code `1124` | on `GetCardBalanceChangeOrder` only: MyPal never received that order |
| any other `*APIError` (`1000`, `1100`, `500`, HTTP 5xx, unknown code) | outcome unknown |
| any error that is not an `*APIError` and not in the first row: timeout, connection reset, redirect, undecodable or oversized body (`ErrInvalidResponse`) | outcome unknown |

When the outcome is unknown, call `GetCardBalanceChangeOrder` with the same `RequestOrderID`, or wait for the webhook. If you resend, resend with the **same** `RequestOrderID` (MyPal answers `1123` for a duplicate), never a new one. The SDK never retries. MyPal rate-limits two writes on the same card within about one second.

### Answer a 3DS request

The `3ds` webhook carries either an OTP to show to the user (`verificationType` `otp`) or an `AuthID` (`http`). For the second, call `Approve3DS` or `Reject3DS`.

## Methods

| Method | What it does |
|---|---|
| `RegisterCustomer`, `GetCustomerUID` | Creates a user; finds a uid by email |
| `ListKYCCountries`, `UploadKYCFile`, `SubmitKYC`, `UpdateKYC`, `GetKYCStatus` | KYC |
| `ListCardProducts`, `ListCards`, `ApplyCard` | Card products, a user's cards, issue a card |
| `CanActivateCard`, `ActivateCard`, `SetCardPIN`, `QueryCardPIN`, `SetCardNoPINLimit` | Physical cards |
| `RechargeCard`, `WithdrawCard`, `GetCardBalanceChangeOrder` | Move money between the merchant wallet and a card |
| `GetCardBalance`, `GetCardInfo`, `UpdateCardStatus`, `CloseCard`, `UpdateCardEmail` | Read and manage a card |
| `ListTransactions`, `GetTransaction`, `Approve3DS`, `Reject3DS` | Transactions and 3DS |
| `ListDeliveryRegions`, `AddDeliveryAddress`, `UpdateDeliveryAddress`, `GetDeliveryInfo` | Mailing of physical cards |
| `GetMerchantAssets` | Merchant wallet balances |
| `Simulate3DS`, `SimulateTrade`, `SimulateRefund`, `SimulateClear`, `SimulateReversal` | Test activity, sandbox only |

Every method returns `*gomypal.Response[T]` with `Code`, `Msg` and `Data`.

The Simulate methods are sandbox-only. They return `gomypal.ErrSandboxOnly` without sending anything when the client points at the production host.

### Money

Amounts, balances, fees and rates are `gomypal.Decimal`, a decimal string such as `"10.5"`. Do not parse them into a float. Where MyPal wants a JSON number, the SDK writes exactly the digits you gave, and signs the same text.

`RechargeCard` and `WithdrawCard` also reject an amount that is 0 or negative or has more than 2 decimals. In requests, pass amounts without trailing zeros (`"10.5"`, not `"10.50"`) and without an exponent. Only that form is proven to pass MyPal's signature check. A malformed amount fails with `gomypal.ErrInvalidDecimal` before the call.

### PIN

`SetCardPIN` takes the encrypted PIN. `gomypal.EncryptPIN(apiAESKey, pin)` and `gomypal.DecryptPIN(apiAESKey, encrypted)` do the AES step with the `api-aesKey` of the merchant dashboard (base64 of 32 bytes, AES-256).

## Webhooks

MyPal posts every event to one URL, with the signature in the `sign` header. Read the raw body, verify, then parse. The SDK ships no HTTP handler.

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
		return
	}

	if err := gomypal.VerifyWebhookSignature(webhookKey, body, r.Header.Get(gomypal.WebhookSignHeader)); err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	event, err := gomypal.ParseWebhookEvent(body)
	if err != nil {
		_ = json.NewEncoder(w).Encode(gomypal.WebhookAckFailure())
		return
	}

	if event.CardRechargeResult != nil {
		// event.CardRechargeResult.OrderID is your RequestOrderID
	}

	_ = json.NewEncoder(w).Encode(gomypal.WebhookAckSuccess())
}
```

- `webhookKey` is the `webhook-aesKey` text exactly as shown in the dashboard. An empty key or an empty signature is rejected.
- MyPal retries up to 7 times (1, 2, 4, 8, 15, 30 and 60 minutes after the event). For money events (`cardRechargeResult`, `cardWithdrawResult`, `merchantRecharge`) de-duplicate on the order id plus the status (or on `TxHash`), not on `eventId`.
- A webhook carries no timestamp, so a captured one can be replayed with its valid signature. Before you credit anything, confirm with `GetCardBalanceChangeOrder`.
- `VerifyWebhookSignature` and `ParseWebhookEvent` return `gomypal.ErrAmbiguousWebhookBody` for a body that the signature also covers in another reading (keys that differ only by case, a quote, `&` or `=` inside an identifier, a text value that imitates a `key=value` pair). Treat it like a bad signature. Keys are read case-sensitively.
- An unknown `eventType` is not an error: every payload pointer is nil and `event.Raw` holds the body.
- Payloads can hold an OTP or card data. Do not log them.

## Errors

A response with HTTP status 400 or above, and a response whose `code` is not 0 (MyPal sends those with HTTP 200 too), is a `*gomypal.APIError`.

A reply the SDK cannot trust is an error wrapping `gomypal.ErrInvalidResponse`, never a success: a status below 400 that is not 2xx (redirects are not followed), a body that is not the `{code,msg,data}` envelope, `data` that does not decode (for endpoints whose response has no fields, `data` is ignored), or a body over 10 MiB (also `ErrResponseTooLarge`).

```go
var apiErr *gomypal.APIError
if errors.As(err, &apiErr) {
	log.Printf("%s (code %d, http %d)", apiErr.Message, apiErr.Code, apiErr.StatusCode)
}

if gomypal.IsNotFound(err) {
	// user, card, order, address or account does not exist
}
```

| Helper | True for |
|---|---|
| `IsNotFound` | HTTP 404; codes 1007, 1102, 1103, 1110, 1124, 1201, 1202, 1204, 1300, 1301, 1501 |
| `IsUnauthorized` | HTTP 401, 403; codes 403, 1002, 1003, 1004, 1005, 1006, 1008 |
| `IsConflict` | HTTP 409; codes 1011, 1108, 1123, 1305, 1401, 1402, 1502 |
| `IsRateLimited` | HTTP 429; code 429 |

Every documented code is a constant (`gomypal.CodeInsufficientBalance`, ...). MyPal responses carry no request id, so `APIError.RequestID` is always empty. `APIError.Body` is the raw body; it can hold card data, so do not log it. When the body is not JSON (a proxy page), `Message` holds its first 200 bytes.

A timeout is `errors.Is(err, context.DeadlineExceeded)`. An empty required field is `errors.Is(err, gomypal.ErrMissingField)`, returned before any request is sent.

## Configuration

```go
client, err := gomypal.NewClient(baseURL, appID, privateKey,
	gomypal.WithTimeout(10*time.Second),
	gomypal.WithLogger(slog.Default()),
	gomypal.WithWebhookKey(webhookKey),
	gomypal.WithPINKey(apiAESKey),
)
```

- The default timeout is 30 seconds per call and applies with a custom HTTP client too.
- Debug logs carry the method, path and status, never the host, headers, bodies, signatures or keys. Printing a `*Client` shows only the base URL.
- The default HTTP client does not follow redirects. If you pass your own with `WithHTTPClient`, set `CheckRedirect` to return `http.ErrUseLastResponse` as well; otherwise the signed headers and body are sent to the redirect target.
- Response bodies over 10 MiB are refused (`gomypal.ErrResponseTooLarge`). `UploadKYCFile` refuses a file over 10 MiB (`gomypal.ErrFileTooLarge`).
- `WithClock` and `WithNonceSource` exist for tests.

## Limitations

- MyPal's signing removes every double quote from the signed string and does not escape `&`, `=`, `,` or `:`. Two bodies that differ only by quotes therefore get the same signature, and the SDK cannot change that. It rejects the ambiguous webhook bodies it can recognise (see Webhooks).
These come from gaps in the MyPal documentation. They are listed in `api-spec.md` ("Open questions").

- No sandbox URL is published. Ask MyPal for it.
- `UploadKYCFile` signs the headers only. MyPal does not document how a multipart request is signed, and the file size limit is unknown. Verify on your sandbox before release.
- Whether the Simulate endpoints need the signing headers and a uid is not documented. The SDK signs them; `UID` is optional there.
- Amounts with trailing zeros or very large values are not proven to pass signature verification.
- The server's tolerance for the timestamp and the rules for the nonce are not documented. Keep the server clock in sync.
- `ListTransactions` returns a bare list without a total. The first page index and the largest page size are not documented.
- `GetCardInfo` reads the card state from `status`, or from `cardStatus` when `status` is absent.
- Webhooks: the signature form for empty values is not confirmed, so the SDK accepts two forms: the vendor Java sample (keeps every key) and the one a production integration uses (drops only `null` keys). Both need the secret key, but accepting two forms means one body can have two valid signatures. For `cardWithdrawResult` the documented example shows `eventType` `cardRechargeResult`; `merchantRecharge` documents the key `curreny` (the SDK reads both spellings); the units of `CreateAt` and `TxTime` and the values of `kycStatusChange.auditState` are not documented.
- Currency of `NoPinLimit` and of delivery `Price`, and the time zone of `OccurTime`, are not documented.

## Tests

```bash
go test ./...
```

Unit tests run against a fake server that checks the RSA signature of every request. They need no network and no keys.

Sandbox tests are skipped unless the variables in `.env.example` are set:

```bash
go test -run TestSandbox_ ./...
```

They never run against production: the test client stops when `MYPAL_BASE_URL` is the production host. On the sandbox they leave behind one registered user per run (email `sdk-sandbox-test+<time>@example.com`; MyPal has no delete-user endpoint) and, when `MYPAL_TEST_UID` and `MYPAL_TEST_CARD_ID` are set, one recharge order `sdk-sandbox-test-<time>` and one simulated transaction on that card. Nothing is cleaned up.

## Versioning

Semantic versioning. The version is `gomypal.Version` and is sent as `User-Agent: gomypal/<version>`.
