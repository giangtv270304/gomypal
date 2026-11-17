# API spec for the SDK: gomypal (MyPal card API)

Source of every statement is the vendor documentation (Apifox project "cards-api-en") unless tagged:

- **[existing]** = observed in existing integrations, not in the docs. "Integration A" and "integration B" are two such clients; "earlier client" is a previous Go client.
- **[user]** = given by the user.
- **[verified]** = recomputed while writing this spec from a worked example printed in the vendor docs.
- **UNCONFIRMED: ...** = docs silent or contradictory; also listed in "Open questions". Never code a guess for these without the stated fallback.

## 1. Identity

| Item | Value |
|---|---|
| SDK name / module | `gomypal` / `github.com/giangtv270304/gomypal` |
| Go package | `package gomypal` (flat; signer in sub-package `signer`) |
| Production base URL | `https://api.cardapi.vip/openapi` **[user]** (docs only show the Apifox placeholder `http://prod-cn.your-api-server.com`) |
| Sandbox base URL | UNCONFIRMED: sandbox base URL. No constant in the SDK; tests read env `MYPAL_BASE_URL` |
| Path form | every path below is appended to the base URL, e.g. `https://api.cardapi.vip/openapi/user/register` |
| `appId` | "Agent platform application obtained" (merchant dashboard). UNCONFIRMED: dashboard URL |
| RSA private key (request signing) | merchant's RSA private key, base64 of PKCS#8 DER, no PEM armor in the docs. UNCONFIRMED: who generates the key pair and where the public key is registered |
| `webhook-aesKey` (webhook HMAC key) | "configured in the merchant's backend webhook settings" |
| `api-aesKey` (PIN encryption key) | "can be obtained from the merchant dashboard" |
| Signature version | "Signature Description v2" (`version: v2`). v1 is documented as "Supported until 2026-07-15", i.e. expired: not implemented |

Client constructor inputs (all required, error at construction when empty): base URL, `appId`, RSA private key. Optional: webhook key, PIN key, HTTP client, logger, timeout (default 30s), clock, nonce source.

## 2. Authentication (Signature Description v2)

### 2.1 Headers

All 37 endpoints are `POST`. JSON endpoints send `Content-Type: application/json`.

| Header | Required | Value |
|---|---|---|
| `appId` | yes | merchant app id |
| `nonce` | yes | "Random numbers". Doc example: `TD4ENK2SN25HQBGPGTS5A1LSHCGZL4XR` (32 chars, `A-Z0-9`). Server rejects a repeated nonce (code `1006`). UNCONFIRMED: allowed length/charset and the replay window. **[existing]** 32 chars `A-Z0-9` (integration B, earlier client) and 24 lowercase hex chars (integration A) are both accepted. SDK: 32 chars `A-Z0-9` from `crypto/rand` |
| `timestamp` | yes | Unix time in **milliseconds**, decimal string (doc example `1764252493756`). Codes `1003` (format) and `1004` (expired) exist. UNCONFIRMED: tolerance window |
| `uid` | per endpoint (see section 4) | MyPal user id returned by Register, e.g. `10001`. Header type string |
| `version` | yes | `v2` ("Default value: v2") |
| `sign` | yes | signature, see below |
| `User-Agent` | SDK | `gomypal/<Version>` |

Header names are case-sensitive in the canonical string only. On the wire Go canonicalises them (`Appid`, `Nonce`...); **[existing]** production integrations send them that way and it works.

Endpoints without `uid`: the canonical string still contains `uid=` (empty). On the wire: **[existing]** integration A and integration B send the header `uid` with an empty value; earlier client omits it. UNCONFIRMED: whether an omitted `uid` header is accepted. SDK: send `uid` with empty value (the form proven in production).

The "07.Simulate" pages declare no header parameters at all (`parameters: []`). UNCONFIRMED: whether simulate endpoints need the signing headers and `uid`. SDK: sign them like every other endpoint; `UID` is optional on simulate requests.

### 2.2 Canonical string

`canonical = headerPart + bodyPart` with **no separator between the two parts** (not even `&`).

**headerPart**: always exactly these five keys, sorted ascending by ASCII, `key=value` joined by `&`. `sign` is excluded. Nothing is skipped, an absent uid gives `uid=`:

```
appId=<appId>&nonce=<nonce>&timestamp=<timestamp>&uid=<uid or empty>&version=<version>
```

**bodyPart**, built from the JSON body object:

| Rule | Detail |
|---|---|
| Fields | top-level fields with a non-empty value ("non-empty fields in the body"). Key `sign` is skipped (Java sample) |
| Sorting | keys ascending by ASCII (Java `TreeMap<String>`; byte order for ASCII keys) |
| Format | `key1=value1&key2=value2`, no trailing `&`, no URL-encoding |
| String value | raw text, no quotes |
| Number value | the exact textual token that is sent in the body (see 2.5) |
| Boolean value | `true` / `false` |
| Nested object | `{k1:v1,k2:v2}`: keys sorted ascending, `:` between key and value, `,` between pairs, no quotes, no spaces. The Java sample supports exactly two levels ("supports two-level objects"). None of the 37 request bodies contains a nested object; the rule is needed for the vector and for webhooks |
| Double quotes | after concatenation **every** `"` character is removed from the whole string (`replaceAll("\"", "")`), including a `"` inside a string value |
| Empty body / no fields | bodyPart is the empty string; canonical is headerPart only. **[existing]** the HTTP body sent is `{}` |
| Empty values | Doc text says only non-empty fields participate; the Java sample (`getSortedString`) does not skip anything. **[existing]** all three integrations skip `null` and `""` and keep `0` and `false`. UNCONFIRMED: server treatment of `""`, `null`, `0`, `false`. SDK: request structs use `omitempty` for optional fields so an empty optional field is never sent; the signer skips `null` and `""`, keeps `0` and `false` |
| Arrays | no request body contains an array. UNCONFIRMED: array canonical form. **[existing]** integration A renders `[v1,v2]`. Java sample would print `JSONArray.toString()` with quotes removed |
| Query parameters | none: no endpoint has query or path parameters. UNCONFIRMED: how a query string would be signed (docs silent). SDK must not add query parameters |
| Multipart upload (`/upload/file`) | UNCONFIRMED: docs do not say how a multipart request is signed. SDK: bodyPart empty (headerPart only, with the uid), flagged for sandbox verification before release |

### 2.3 Algorithm and encoding

| Item | Value |
|---|---|
| Algorithm | Java `SHA1WithRSA` = RSASSA-PKCS1-v1_5 with SHA-1 over the canonical string bytes |
| String bytes | Java sample uses `data.getBytes()` (platform charset). SDK: UTF-8 |
| Private key input | base64 (standard) of PKCS#8 DER (`PKCS8EncodedKeySpec`). SDK also accepts a PEM block; **[existing]** services also accept PKCS#1 DER |
| Signature encoding | base64 standard alphabet **with** padding (`Base64.getEncoder()`), sent in header `sign` |

SDK rules (skill section 4): encode the body once, build the canonical string from those exact bytes (parse them with `json.Decoder.UseNumber` / `json.RawMessage`, never through `float64`), send the same bytes. Clock (`now`) and nonce source are replaceable so tests produce a fixed signature.

### 2.4 Test vector (worked example from the docs, **[verified]** by `signer_test.go` of the earlier client and by `openssl dgst -sha1 -sign`)

The private key below is the test key printed in the vendor page "Signature Description v2". It is the only key allowed in `signer/signature-vectors.json`.

| Input | Value |
|---|---|
| `appId` | `3a9ea1263e624adf88518d4d88369aad` |
| `nonce` | `TD4ENK2SN25HQBGPGTS5A1LSHCGZL4XR` |
| `timestamp` | `1764252493756` |
| `uid` | `102130` |
| `version` | `v2` |

Body:

```json
{"productId":100004,"transaction":{"transType":170,"merchantName":"mypal","balance":"100"},"email":"zhangshuaishuai9527@gmail.com","userBankcardId":100168}
```

Canonical string:

```
appId=3a9ea1263e624adf88518d4d88369aad&nonce=TD4ENK2SN25HQBGPGTS5A1LSHCGZL4XR&timestamp=1764252493756&uid=102130&version=v2email=zhangshuaishuai9527@gmail.com&productId=100004&transaction={balance:100,merchantName:mypal,transType:170}&userBankcardId=100168
```

Private key (vendor doc test key, base64 PKCS#8):

```
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCROf/gagyT3VVEhjelieoqvsEe4QW6KIE+hxXpIGCXYtO2w38rk5ht0VYWLX4dtmFf3vhkzIr2BzRe5cTetKbfPyfLSPhuyxnaSmJC0CVhvQpYsQWnDUQO1xp9N2BrOsW2sN5TTsmqklxZkVFiY4wn1QNo5RyrGQoE8coRvFO5tGcC7ygWuLFy/OQkZ31xOp6DdG3wBXYX/Ok+mjDBfgfZvNwV5mw5y7efoLbZQjlL9P5EYY8m6iSFw0nhZHT1a1gCZKk3lZlNhSPJ8z+KUOTAHV2qoQgygB0HvcIAuR5mFFp62jqRUE2HNRg9nWO5t/x84ShSMradNnRqgV8SiXr9AgMBAAECggEAPx8LdToCz34DIq6Hnz7LddrFS6uYdqpGSqwiPNqiwQnmZWihbIpRVWIIXclWKInt1uKt6fnxaDAHdju+DuJMKXpkzHmh6V4DlZqBDPM1XD7zMUt33nzomN6rl8Mq+FBqQRas+16iTq4gz5lW20HTWDjAq6XxiPhwH2gMYRsO9QaoilzTxpgeFqGBj1DUDDdGyJjdk+ibI3WXazeEEse20x+D7aK4IeGAMbCDHxkw3r1Bcm/3OqVrsMi9ex1V1YFfF1ElFeuHPEhUjCWMdUthWdHuZeIpMrAeNUyYbXzt60x0aDdSNFkj6DdRybhxxouJdMA9ccqU/5khzxpbhfrxTQKBgQDYT1aBWCfTD4cQt+4YqDBaqhBQBeR7Vb3xvqpkiS0rrp9qUExjHPiITI2r1GKnu0vLnw+V+eMcaJZcjVXy6DFjHC9wJEaGhnjkTKDvNw7jpnOikkE4/dT6m2K9MSUooNXtO0ViwenDQ2hjhuDuWf/JpOKqt58jBWUJ6PPAWctbiwKBgQCr368i06IASlqMTP0UZQA1WFH89ejM2gT91mKSCxOGqnFCKledBv1HuRcwFe/qpY4U8iXgT9Q5XvLnRbSuegzmBO7XNp99Qmjs5uk/hUXhhsYekSp/JjSMqoDyeer6s9B4nr/J+1qRHm3KzdAsAvgNdFaCZxaxRq4383bb28L0lwKBgQC5OMaEHy6KxkMLh9PzfSFaoUEypLkO+06YkIFrcSMiwnZzHKssO4R9gb/WIRkRM4olo+A+tilJDC2ktsmgJD8V9O6SAh3512/xXcR40haHzOI7lS0QVhpnNWMJSendrktAxn4VoKJAt5nNj5NPxfUHf48wS+eSn23zXGxMixddHQKBgAU6CzBq8S1DFZfYq4WYQLZelje0Hr1EEAzMHTlFvvVEb6dMdFehI8ivl0XuFNq06Co45u/4XYLVIVvpIcKAka3ZDo5a6eS8RUaXCxbUbbibCaFaVsmBFu4ELWGyVXQETd2tLjXtP+bT3xDgy2jbOkHmfM8AZY9Ej/3BmKa8+yPHAoGAdTn8MPdT4tOcFulIBgQ8Mljc3OAi4xkAZpa7dhgdVOJFIE/Gcj33ioe4loUO8M/EimWB1rdZUCZpxI+ZMwyWjzg5UU+uGktMVdDJ89EhHReDvyIo7mPez0JrhkumtsoUxUSGy/V0RQ9jXYvOPyCY/BUvgj7eaw1obhYU3/+rpvk=
```

Expected `sign`:

```
Z9M/W1aUJhtI1O21DzGKTGW1c44EPWxgeDE1t0/FbHx+ddLrGe5809NEmaDJa//vj2n+P/mx01XTEre36BDGDbgIBkufudM2tMUrxXH3xtb83pVSXa1HFvpgHK73zhBqYxgPm5qLQCFDRJMngvucMAQbEctlNzjBzlAgKq6WcC42+/F3/9JfMYPvgzArBoSnLxMsucRyay5OT1D5iJmcGa0FwfE9g8IsCCsfMHbycc6BHTSsRlOCIrxIX3CCmIHxAH5RvV4xxuKOqG5+WafDtkKn9PvvfvugkrFVlIR/xxInVCae5W4LRisuuxECpQ7ReTqUx6dCwUzO/j39d65TkQ==
```

Second vector (header part only, from the doc's "Signature Notes"; no expected signature is printed, so the vector checks the canonical string only): same `appId`, `nonce`, `timestamp`, `version`, no uid, empty body gives

```
appId=3a9ea1263e624adf88518d4d88369aad&nonce=TD4ENK2SN25HQBGPGTS5A1LSHCGZL4XR&timestamp=1764252493756&uid=&version=v2
```

`signer/signature-vectors.json` fields per vector: `name`, `appId`, `nonce`, `timestamp`, `uid`, `version`, `body` (exact JSON text), `canonical`, `privateKey`, `signature` (empty when only the canonical string is checked).

### 2.5 Numbers in the signed string

The canonical string must contain exactly the same text for a number as the JSON body that is sent: body `"amount":100.5` gives `amount=100.5`. The SDK therefore never formats a number twice. UNCONFIRMED: whether the server re-serialises numbers before verifying (for example `100.00` to `100`, or exponent forms). **[existing]** production sends `float64` amounts, i.e. the shortest form without trailing zeros and without exponent (`100`, `10.5`); only that form is proven. See section 6.

### 2.6 PIN encryption (`pin` in Set Pin / Query PIN)

| Item | Value |
|---|---|
| Cipher | `AES/CBC/PKCS5Padding` |
| Key | base64-decode of `api-aesKey`: exactly 32 bytes (AES-256); the SDK rejects other lengths with `ErrInvalidPINKey` |
| IV | 16 random bytes |
| Output | base64 standard of `IV || ciphertext` |
| Decrypt | base64-decode, first 16 bytes are the IV |
| Vector **[verified]** (decrypts to the PIN) | PIN `123456`, `api-aesKey` `z+7ruiMj0GottZjhNVJG2ZJsDcXqR2b3iW32+Hav34c=`, encrypted `dsOiXZoeUUBgjw5YujQlvNzMtULGKy3vZcoLgiXgfVc=` |

SDK: `EncryptPIN(apiAESKey, pin string) (string, error)` in `card_set_pin.go`, `DecryptPIN(apiAESKey, encrypted string) (string, error)` in `card_query_pin.go`. Test decrypts the vector (encryption is randomised by the IV).

## 3. Response shapes

### 3.1 Envelope

The endpoint pages document only the content of `data`. The envelope itself is **[existing]** (both services decode it in production):

```json
{"code": 0, "msg": "success", "data": { "uid": 1000001 }}
```

| Field | Type | Notes |
|---|---|---|
| `code` | integer | `0` = success; anything else is an error (table 3.3) |
| `msg` | string | message |
| `data` | object, array or null | endpoint payload. Array for the list endpoints. **[existing]** may be `null`/absent for endpoints with an empty response |

Examples:

| Case | Body | Source |
|---|---|---|
| Success, object | `{"code":0,"msg":"success","data":{"uid":1000001}}` | shape **[existing]**; `data` from docs. UNCONFIRMED: exact `msg` text on success ("success" per error-code table) |
| Success, list | `{"code":0,"msg":"success","data":[{"accountId":100387,"accountType":"MAIN","currency":"USDT","availableAmount":"10","frozenAmount":"1"}]}` | same |
| Business error | `{"code":1124,"msg":"order does not exist"}` | **[existing]** test fixture of integration A |
| Rate limit in body | `{"code":429,"msg":"Operation too frequent"}` | **[existing]** test fixture of integration A |

There is no request-id field in any documented or observed response. `APIError.RequestID` stays empty.

SDK type in `response.go`: `Response[T any] struct { Code int64; Msg string; Data T }`. Every method returns `(*Response[XResponse], error)`.

### 3.2 How errors are reported

| Situation | SDK behaviour |
|---|---|
| HTTP 2xx and `code == 0` | success |
| HTTP 2xx and `code != 0` | `*APIError{StatusCode: <http status>, Code: <code>, Message: <msg>, Body: <raw>}`. **Mandatory**: a non-zero `code` is an error even on HTTP 200 |
| HTTP >= 400 | `*APIError`; `Code`/`Message` filled from the body when it is the JSON envelope, otherwise `Code = 0` and `Message` = raw body (HTML proxy pages are kept in `Body`) |
| HTTP 2xx, body not decodable, no `code`, or `data` not decodable into a type with fields | error wrapping `ErrInvalidResponse` (never `nil, nil`). `data` is ignored for responses without fields; `null` or absent `data` gives the zero value |
| HTTP below 400 and not 2xx (3xx: redirects are not followed; 1xx) | error wrapping `ErrInvalidResponse`, never success |
| Body over 10 MiB | error wrapping `ErrInvalidResponse` and `ErrResponseTooLarge` |
| Transport error | wrapped with `%w` |

For money-moving calls (`RechargeCard`, `WithdrawCard`) every error that is not an `*APIError` and not a pre-send validation error (`ErrMissingField`, `ErrInvalidDecimal`, `ErrNilRequest`) means the outcome is unknown.

UNCONFIRMED: which HTTP status accompanies a non-zero `code` (docs silent). **[existing]** integration A handles both HTTP 429 and `code` 429 in a 200 body, and HTTP 4xx/5xx with or without an envelope.

`APIError.Code` is `int64` (codes are numeric). Error text: `gomypal: api error: op=<method> status=<http> code=<code> message=<msg>`.

Helpers (use `errors.As`):

| Helper | True when |
|---|---|
| `IsRateLimited` | HTTP 429 or `code` 429 |
| `IsUnauthorized` | HTTP 401/403 or `code` in 403, 1002, 1003, 1004, 1005, 1006, 1008 |
| `IsNotFound` | HTTP 404 or `code` in 1007, 1102, 1103, 1110, 1124, 1201, 1202, 1204, 1300, 1301, 1501 |
| `IsConflict` | HTTP 409 or `code` in 1011, 1108, 1123, 1305, 1401, 1402, 1502 |

**[existing]** semantics used in production (document in README, do not hard-code retries in the SDK): `1104`, `1105`, `429` prove a recharge or withdraw was not processed; `1124` means a balance-change order query found no such order (not a verdict on a send); by their meaning in the code table `1001`-`1006`, `1008`, `1014`, `403` are rejections before processing (not proven); `1123` means MyPal already holds this `requestOrderId`, so a resend must keep the same `requestOrderId`; a timeout, HTTP 5xx/408/499, `1000`, `1100` or an unknown code is ambiguous for money-moving calls: query `/bankcard/balance/change/order` or wait for the webhook. MyPal rate-limits two writes on the same card within about one second.

### 3.3 Error codes (page "Error code", complete)

| Code | Message | Code | Message |
|---|---|---|---|
| 0 | success | 1114 | card upstream error |
| 403 | access denied | 1115 | card upstream timeout |
| 429 | too many requests, please try again later | 1116 | card upstream unavailable |
| 500 | system error | 1117 | card channel configuration error |
| 1000 | business processing failed | 1118 | card channel authentication failed |
| 1001 | params validation failed | 1119 | card channel authorization failed |
| 1002 | wrong appId | 1120 | insufficient USD balance for card application |
| 1003 | timestamp format error | 1121 | insufficient USDT balance for card application |
| 1004 | authorized signature time has expired | 1122 | kyc level does not meet requirements |
| 1005 | signature verification failed | 1123 | duplicate card request |
| 1006 | nonce repeated | 1124 | order does not exist |
| 1007 | the user has not registered | 1200 | delivery service error |
| 1008 | the current user not belong you | 1201 | delivery region not found |
| 1009 | email format error | 1202 | delivery address not found |
| 1010 | wrong product id | 1203 | delivery address invalid |
| 1011 | The email has registered | 1204 | delivery order not found |
| 1012 | image format error | 1205 | delivery order does not belong to current user |
| 1013 | no card in possession | 1300 | merchant sub-account not found |
| 1014 | uid is required | 1301 | merchant main account not found |
| 1100 | card service error | 1302 | merchant account parameter error |
| 1101 | card parameter error | 1303 | merchant account insufficient balance |
| 1102 | card not found | 1304 | merchant account recharge failed |
| 1103 | cardholder not found | 1305 | duplicate merchant account request |
| 1104 | The card is not activated; please try again after activation. | 1400 | kyc application failed |
| 1105 | insufficient balance | 1401 | kyc already approved |
| 1106 | card transaction failed | 1402 | kyc application already exists |
| 1107 | kyc required | 1403 | kyc phone number invalid |
| 1108 | card already bound | 1404 | kyc name invalid |
| 1109 | card permission denied | 1405 | kyc document required |
| 1110 | kyc information not found | 1406 | kyc status query failed |
| 1111 | card operation not supported | 1500 | customer registration failed |
| 1112 | card channel network error | 1501 | customer merchant not found |
| 1113 | card channel timeout | 1502 | duplicate customer registration request |

Export every code as a typed constant (`ErrorCode` consts in `errors.go`). An unknown code must not break decoding.

## 4. Endpoints (37)

Conventions for the tables:

- Every endpoint is `POST <base><path>`. JSON body unless stated. No path or query parameters anywhere.
- Signature: `func (c *Client) <Method>(ctx context.Context, req *<Method>Request) (*Response[<Method>Response], error)`. Each file holds `<Method>Request`, `<Method>Response` and the method.
- `uid` column: `yes` = header required (docs); the request struct carries it as `UID string` with tag `json:"-"`, and an empty `UID` is an error before the call. `no` = header sent empty. `?` = UNCONFIRMED (simulate), optional `UID`.
- Field notation: `name: type` required, `name?: type` optional (`omitempty`). Types: `str`, `int` (JSON integer, Go `int64`), `bool`, `money` (section 6), `id` (see 5.1).
- Response column describes `data`. `{}` = empty object (SDK response struct is empty; tolerate `null`). `[]T` = `data` is a JSON array.

### 01.Customer

| # | Method | Path | SDK method | File | uid | Request body | Response `data` | Notes |
|---|---|---|---|---|---|---|---|---|
| 1 | POST | `/user/register` | `RegisterCustomer` | `customer_register.go` | no | `email: str` ("Must be unique") | `uid: int` | errors `1009`, `1011`, `1500`, `1502` |
| 2 | POST | `/user/uid` | `GetCustomerUID` | `customer_get_uid.go` | no | `email: str` | `uid: int` | "Query registered user UID via email"; `1007` when not registered |

### 02.KYC

| # | Method | Path | SDK method | File | uid | Request body | Response `data` | Notes |
|---|---|---|---|---|---|---|---|---|
| 3 | POST | `/user/kyc/country/list` | `ListKYCCountries` | `kyc_country_list.go` | no | `name?: str` (filter; all countries when blank) | `[]KYCCountry` | |
| 4 | POST | `/upload/file` | `UploadKYCFile` | `kyc_file_upload.go` | yes | **multipart/form-data**, one part: field name `idCard`, binary file, allowed `.png`, `.pdf`, `.jpg`, `.jpeg` | `fileUrl: str` | see "File upload" below |
| 5 | POST | `/user/kyc/apply` | `SubmitKYC` | `kyc_submit.go` | yes | `KYCInfo` (5.3) | `{}` | asynchronous review; poll #7 |
| 6 | POST | `/user/kyc/update` | `UpdateKYC` | `kyc_update.go` | yes | `KYCInfo` (identical schema and required list to #5) | `{}` | page description is copied from #5 |
| 7 | POST | `/user/kyc/status` | `GetKYCStatus` | `kyc_get_status.go` | yes | empty (`{}`) | `status: KYCStatus`, `failedReason?: str` | doc example `{"status":"success","failedReason":""}` |

File upload (#4), exact details:

| Item | Value |
|---|---|
| Content-Type | `multipart/form-data; boundary=...` |
| Form field | `idCard` (required, `format: binary`). No other form fields documented |
| File types | `.png`, `.pdf`, `.jpg`, `.jpeg`; code `1012` "image format error" |
| Headers | the six signing headers including `uid` |
| Signing | UNCONFIRMED (2.2): SDK signs headerPart only |
| Limits | UNCONFIRMED: maximum file size; whether the part needs a `filename` and a specific part `Content-Type` |
| SDK request | `UID string`, `FileName string` (required, used as part filename), `Content io.Reader` (required, at most 10 MiB read into memory, `ErrFileTooLarge` above) |
| Response | `fileUrl: str`, used as `idUrl`, `idBackUrl`, `fileUrl`, `faceUrl`, `selfieUrl` in `KYCInfo` |

### 03.Card

| # | Method | Path | SDK method | File | uid | Request body | Response `data` | Notes |
|---|---|---|---|---|---|---|---|---|
| 8 | POST | `/bankcard/merchant/template/list` | `ListCardProducts` | `card_product_list.go` | no | empty (`{}`) | `[]CardProduct` | |
| 9 | POST | `/bankcard/user/card/list` | `ListCards` | `card_list.go` | yes | empty (`{}`) | `[]UserCard` | |
| 10 | POST | `/bankcard/apply` | `ApplyCard` | `card_apply.go` | yes | `productId: int`, `deliveryAddressId?: int` (required for a physical card) | `userBankcardId: int`, `orderNo: str` | **[existing]** integration A also reads `cardNo: str` (not in docs): decode as optional. Errors `1010`, `1107`, `1120`, `1121`, `1122`, `1123` |
| 11 | POST | `/bankcard/canActive` | `CanActivateCard` | `card_can_activate.go` | yes | `cardNo?: str`, `verifyCode?: str` ("cannot both be empty"), `cvv?: str`, `expDate?: str` (`08/28`, month/year) | `canActive: bool` | physical cards. SDK rejects the call when both `cardNo` and `verifyCode` are empty |
| 12 | POST | `/bankcard/active` | `ActivateCard` | `card_activate.go` | yes | `productId: int`, `mobilePrefix: str`, `mobile: str`, `countryCode: str` (ISO 3166-1 alpha-3), `address: str`, `city: str`, `state: str`, `postCode: str`, `cardNo?: str`, `verifyCode?: str` (not both empty), `address2?: str`, `cvv?: str`, `expDate?: str` (`08/28`) | `userBankcardId: id` (schema string, example integer `1000`) | physical cards |
| 13 | POST | `/bankcard/setPin` | `SetCardPIN` | `card_set_pin.go` | yes | `userBankcardId: int`, `pin: str` (AES-encrypted, 2.6) | `{}` | physical cards. Request takes the already encrypted value; `EncryptPIN` lives in this file |
| 14 | POST | `/bankcard/recharge` | `RechargeCard` | `card_recharge.go` | yes | `userBankcardId: int`, `amount: money` (JSON number, max 2 decimals), `requestOrderId: str` (must be unique) | `{}` | SDK rejects an amount that is not above 0 or has more than 2 decimals before sending. Asynchronous: result by webhook `cardRechargeResult` or #16. Errors `1104`, `1105`, `1123` |
| 15 | POST | `/bankcard/withdraw` | `WithdrawCard` | `card_withdraw.go` | yes | `userBankcardId: int`, `amount: money` (JSON number, max 2 decimals), `requestOrderId: str` (must be unique) | `{}` | SDK rejects an amount that is not above 0 or has more than 2 decimals before sending. Moves card balance to the merchant wallet. Result by webhook `cardWithdrawResult` or #16 |
| 16 | POST | `/bankcard/balance/change/order` | `GetCardBalanceChangeOrder` | `card_get_balance_change_order.go` | yes | `requestOrderId: str`, `type: BalanceChangeType` | `requestOrderId: str`, `status: OrderStatus` | `1124` when MyPal never received the order. **[existing]** `data` may be null |
| 17 | POST | `/bankcard/balance` | `GetCardBalance` | `card_get_balance.go` | yes | `userBankcardId: int` | `balance: money` (string), `currency: str`, `symbol: str`, `cardLimit?: money` (string) | example `{"balance":"1000","currency":"USD","symbol":"USD","cardLimit":"1200"}` |
| 18 | POST | `/bankcard/update/status` | `UpdateCardStatus` | `card_update_status.go` | yes | `userBankcardId: int`, `enable: bool` (`true` = unfreeze, `false` = freeze) | `{}` | `enable` is required and must be sent when `false`: no `omitempty` |
| 19 | POST | `/bankcard/close` | `CloseCard` | `card_close.go` | yes | `userBankcardId: str` (schema string; example integer `100`) | `{}` | the schema also lists a stray property `01KAZNHCVQRFQ69QKA61AQNDWN`: ignore. **[existing]** sent as a JSON string |
| 20 | POST | `/bankcard/info` | `GetCardInfo` | `card_get_info.go` | yes | `userBankcardId: int` | `infoType: int` (`0` = url, `1` = plain text), `status: CardStatus`, `cardNumber?: str`, `cvv?: str`, `expireDate?: str` (example `2030-08`), `cardPanUrl?: str` | `cardNumber`, `cvv`, `expireDate` only when `infoType=1`; when `infoType=0` fetch them from `cardPanUrl`. Schema field is `status`, the example shows `cardStatus`: UNCONFIRMED which key is returned; **[existing]** reads `status`. SDK decodes both. Response holds PAN/CVV: never log |
| 21 | POST | `/bankcard/update/email` | `UpdateCardEmail` | `card_update_email.go` | yes | `email: str`, `userBankcardId: str` (schema string; example integer) | `{}` | "supported for select card BINs" (`1111` otherwise) |
| 22 | POST | `/bankcard/queryPin` | `QueryCardPIN` | `card_query_pin.go` | yes | `userBankcardId: str` (schema string; example integer) | `pin: str` (AES-encrypted, 2.6) | physical cards, select BINs. `DecryptPIN` lives in this file |
| 23 | POST | `/bankcard/setNoPinLimit` | `SetCardNoPINLimit` | `card_set_no_pin_limit.go` | yes | `userBankcardId: int`, `noPinLimit: money` (JSON number) | `{}` | "physical card password-free limit". UNCONFIRMED: currency/unit of `noPinLimit` |

### 04.Transactions

| # | Method | Path | SDK method | File | uid | Request body | Response `data` | Notes |
|---|---|---|---|---|---|---|---|---|
| 24 | POST | `/bankcard/trade/details` | `ListTransactions` | `transaction_list.go` | yes | `userBankcardId: int`, `pageSize?: int` (default 10), `pageNum?: int` (default first page), `startTime?: int` (ms), `endTime?: int` (ms) | `[]Transaction` | bare array, no total/paging metadata documented. UNCONFIRMED: whether `pageNum` starts at 1 (example uses 1) and the maximum `pageSize` |
| 25 | POST | `/bankcard/trade/info` | `GetTransaction` | `transaction_get.go` | yes | `id: int` (transaction record id, `Transaction.id`) | `Transaction` | shared type lives here |
| 26 | POST | `/bankcard/3ds/approve` | `Approve3DS` | `transaction_3ds_approve.go` | yes | `authId: str` (from webhook `3ds`, `verificationType=http`) | `{}` | |
| 27 | POST | `/bankcard/3ds/reject` | `Reject3DS` | `transaction_3ds_reject.go` | yes | `authId: str` | `{}` | |

### 05.Delivery

| # | Method | Path | SDK method | File | uid | Request body | Response `data` | Notes |
|---|---|---|---|---|---|---|---|---|
| 28 | POST | `/delivery/region` | `ListDeliveryRegions` | `delivery_region_list.go` | no | `local: Locale` (required) | `[]DeliveryRegion` | |
| 29 | POST | `/delivery/address/add` | `AddDeliveryAddress` | `delivery_address_add.go` | yes | `countryRegionId: int`, `country: str`, `city: str`, `receiverName: str`, `receiverMobile: str`, `receiverAddress: str`, `postCode: str` | `id: int` (user address id, used as `deliveryAddressId` in #10) | all fields required |
| 30 | POST | `/delivery/address/update` | `UpdateDeliveryAddress` | `delivery_address_update.go` | yes | `id: id`, `countryRegionId: id` (both schema string, example integers `1000001`, `3`), `country: str`, `city: str`, `receiverName: str`, `receiverMobile: str`, `receiverAddress: str`, `postCode: str` | `id: id` (schema string, example integer) | all fields required. UNCONFIRMED: wire type of `id`/`countryRegionId`; SDK sends JSON integers as in the doc example and as #29 |
| 31 | POST | `/delivery/info` | `GetDeliveryInfo` | `delivery_get_info.go` | yes | `orderNo: str` (from #10) | `deliveryStatus: DeliveryStatus`, `deliveryNo: str` | errors `1204`, `1205` |

### 06.Merchant

| # | Method | Path | SDK method | File | uid | Request body | Response `data` | Notes |
|---|---|---|---|---|---|---|---|---|
| 32 | POST | `/merchant/balance` | `GetMerchantAssets` | `merchant_get_assets.go` | no | none documented. **[existing]** `{}` is sent | `[]MerchantAsset` | |

### 07.Simulate Transactions (sandbox only)

These endpoints exist to generate test activity. They must never be called against production: each method returns `ErrSandboxOnly` before sending when the client base URL host equals the host of the production constant (compared lower-case, without a trailing dot, any path or port).

| # | Method | Path | SDK method | File | uid | Request body | Response `data` | Notes |
|---|---|---|---|---|---|---|---|---|
| 33 | POST | `/bankcard/simulate/3ds` | `Simulate3DS` | `simulate_3ds.go` | ? | `bankCardId: int`, `transactionAmount: money` (number), `transactionCurrency: str` | `{}` | triggers webhook `3ds` |
| 34 | POST | `/bankcard/simulate/auth` | `SimulateTrade` | `simulate_trade.go` | ? | `bankCardId: int`, `transactionAmount: money` (number), `transactionCurrency: str`, `billAmount: money` (number) | `{}` | UNCONFIRMED: how the caller learns the resulting `transactionId` (response is empty; presumably webhook `transactionCreated` or #24) |
| 35 | POST | `/bankcard/simulate/refund` | `SimulateRefund` | `simulate_refund.go` | ? | `transactionId: str`, `transactionAmount: money` (number), `billAmount: money` (number) | `{}` | |
| 36 | POST | `/bankcard/simulate/clear` | `SimulateClear` | `simulate_clear.go` | ? | `transactionId: str`, `billAmount: money` (number), `transactionAmount: money` (number), `transactionCurrency: str` | `{}` | |
| 37 | POST | `/bankcard/simulate/reversal` | `SimulateReversal` | `simulate_reversal.go` | ? | `transactionId: str`, `transactionAmount: money` (number), `billAmount: money` (number) | `{}` | "simulate order cancellation" |

Note: the card id field is `bankCardId` in simulate requests and `userBankcardId` everywhere else. UNCONFIRMED: that both carry the same id.

## 5. Types

### 5.1 Ids

The docs are inconsistent about id wire types (integer in one page, string in another, integer examples under string schemas). Rules:

| Direction | Rule |
|---|---|
| Request | send the wire type listed in section 4 (`int` = JSON number, `str` = JSON string). Go field type is `int64` for `int` and `string` for `str` |
| Response and webhook | fields marked `id` use the named type `ID` (in `response.go`): Go `int64` underneath, unmarshals from a JSON number **or** a numeric string, marshals as a number. **[existing]** integration A needs this (`flexInt`) for `userBankcardId` and `productId` in the card list |
| `uid` | integer in Register/Query User responses (`int64`), string in the header (`UID string` on requests) |

### 5.2 Shared objects

| Type | Defined in | Fields |
|---|---|---|
| `KYCCountry` | `kyc_country_list.go` | `name: str` (country name, e.g. `中国大陆`), `enName: str`, `value: str` (alpha-3, e.g. `CHN`), `profValue: str` (alpha-2), `code: str` (numeric 3-digit, e.g. `156`), `telephoneCode: str` (e.g. `86`), `needCertificate: bool` (extra documents needed: send `fileType`/`fileUrl`/`faceUrl`) |
| `KYCInfo` | `kyc_submit.go` | see 5.3 |
| `CardProduct` | `card_product_list.go` | `id: int` (product id), `bankCardNature: CardType`, `ccy: str`, `applyFee: money`, `rechargeFee: money` (rate between 0 and 1), `bankcardRegion: str`, `cardBin: str`, `activeMinLimit: money`, `rechargeMinLimit: money`, `cardBrand: str`, `cardMode: CardMode`. Example also has `title: str` (not in schema; decode as optional) and lacks `cardBrand`/`cardMode` although "required": treat both as optional |
| `UserCard` | `card_list.go` | `userBankcardId: id`, `cardType: CardType`, `productId: id`, `cardNumber: str`, `status: CardStatus`, `currency: str` |
| `Transaction` | `transaction_get.go` | `id: int`, `userBankcardId: int`, `localCurrencyAmt: money` (bill amount), `localCurrency: str`, `recordNo: str`, `occurTime: str` (`2025-11-28 18:53:07`; keep as string, UNCONFIRMED: time zone), `respCodeDesc: str`, `feeAmount: money`, `feeCurrency: str`, `merchantName: str`, `transType: TransType`, `transStatus: TransStatus`, `transactionId: str`, `originalTransactionId?: str`, `transCurrencyAmt?: money`, `transCurrency?: str`, `merchantCategoryCode?: str`. The last three are documented only for the list (#24) but **[existing]** are decoded for both. List example also shows `accountNum: str` and `failureReason: str` (not in schema): decode as optional. `userBankcardId` is missing from the list example |
| `DeliveryRegion` | `delivery_region_list.go` | `id: id` (schema string, example integer), `countryRegion: str`, `countryRegionZh: str`, `nationalCode: str` (e.g. `0213`), `price: money` (string, postage; UNCONFIRMED: currency) |
| `MerchantAsset` | `merchant_get_assets.go` | `currency: str`, `availableAmount: money` (string), `frozenAmount: money` (string), `accountId: id` (schema string, example integer), `accountType: AccountType` |

### 5.3 `KYCInfo` (body of Submit and Update KYC)

| Field | Type | Required | Notes |
|---|---|---|---|
| `firstName` | str | yes | English letters only |
| `lastName` | str | yes | English letters only |
| `idNo` | str | yes | ID number |
| `email` | str | yes | |
| `nationCode` | str | yes | nationality, ISO 3166-1 alpha-3 (e.g. `SGP`) |
| `certType` | `CertType` (int) | yes | |
| `idUrl` | str | yes | ID photo front URL (from #4) |
| `idBackUrl` | str | no | back photo; needed for cert types 1 and 3 |
| `birthday` | str | yes | example `2025-11-26` (`YYYY-MM-DD`) |
| `countryCode` | str | yes | country of residence, alpha-3 |
| `areaCode` | str | yes | phone area code, e.g. `86` |
| `phone` | str | yes | |
| `fileType` | `KYCFileType` (int) | no | only when the country list says `needCertificate` |
| `fileUrl` | str | no | only with `needCertificate` |
| `faceUrl` | str | no | face report URL, only with `needCertificate` |
| `selfieUrl` | str | no | face selfie URL |
| `referenceId` | str | no | session id of the third-party face report |
| `referenceType` | str | no | third-party report type, e.g. `SUMSUB` |

### 5.4 Enums

Go: named type plus constants; an unknown value must decode without error.

| Enum (Go type, base) | Values |
|---|---|
| `KYCStatus` (string) | `uncommitted` (not submitted), `waiting` (pending review), `success` (approved), `fail` (review failed), `wait_confirm` (pending user confirmation), `wait_audit` (under review) |
| `CertType` (int) | `1` ID card (front and back photos), `2` passport (front only), `3` driver's license (front and back) |
| `KYCFileType` (int) | `7` e-visa, `8` visa photo, `9` Permanent Resident Card (PR), `10` Temporary Residence Permit (TRC), `11` Employment Permit Card, `12` Student Residence Permit, `13` other, `14` electricity/water/gas bills, `15` driver's license / residence permit / ID card (including address), `16` bank statement / credit card statement, `17` property certificate, `18` tax documents / municipal tax bills / government mail, `19` social security card (including address), `20` house rental agreement |
| `CardType` (string) | `VIRTUAL`, `PHYSICAL` (fields `bankCardNature`, `cardType`) |
| `CardMode` (string) | `NORMAL` (normal card), `SHARE` (share card) |
| `CardStatus` (int) | `0` initial state, `1` pending activation, `2` activation in progress, `3` normal, `4` freeze, `5` loss report, `6` before cancellation, `7` cancellation, `8` expired, `9` pause |
| `CardInfoType` (int) | `0` url, `1` plain text |
| `BalanceChangeType` (string) | `RECHARGE` (card top-up), `WITHDRAW` (card withdraw) |
| `OrderStatus` (string) | `PENDING`, `SUCCESS`, `FAILURE` (query #16; the webhooks use only `SUCCESS`, `FAILURE`) |
| `TransType` (int) | `170` card consumption, `174` card pre-authorization, `171` card refund, `172` card transaction fees, `173` card ATM cash withdrawal, `140` apply for a card, `150` card top-up, `175` card cancellation, `165` card withdrawal. Doc examples show `202`, which is not in the enum |
| `TransStatus` (int) | `0` initialization, `1` processing, `2` success, `3` fail |
| `Locale` (string) | `zh_CN` (Simplified Chinese), `zh_HK` (Traditional Chinese), `en_US` (English), `ko_KR` (Korean) |
| `DeliveryStatus` (string) | `NOT_DELIVERY` (to be mailed), `DELIVERY` (mailed) |
| `AccountType` (string) | `MAIN` (main account), `SUB` (sub account) |
| `WebhookEventType` (string) | `3ds`, `cardStatusChange`, `cardRechargeResult`, `cardWithdrawResult`, `transactionCreated`, `merchantRecharge`, `merchantTradeFee`; plus `kycStatusChange` (only in the Overview example) |
| `VerificationType` (string) | `otp` (verification code in `otp`), `http` (approve or reject with `authId`). Description text writes `OTP`/`HTTP`, enum is lower case |
| `ThreeDSScene` (string) | `verification` (card-linking verification code), `3ds` (transaction 3DS verification code) |
| `CardStatusEvent` (string) | `cardFreeze`, `cardActive`, `cardClose`, `cardPreCancelled`. **[existing]** mapping to `CardStatus`: `cardActive`=3, `cardFreeze`=4, `cardClose`=7, `cardPreCancelled`=6 |

## 6. Money

SDK rule: money is a decimal string in Go, never `float64`. `money.go` defines `type Decimal string`:

| Behaviour | Rule |
|---|---|
| Marshal | raw JSON number with exactly the text held (`Decimal("100.5")` gives `100.5`), no quotes. Used for request fields whose wire type is a JSON number |
| Unmarshal | accepts a JSON number (text kept verbatim via the raw token), a JSON string (`"7.00"`), `""` and `null` (both give empty `Decimal`). Never goes through `float64` |
| Validation | on marshal the value must match `^-?[0-9]+(\.[0-9]+)?$` (no exponent, no empty value for a required field); otherwise the call fails before sending |
| Signing | the signer reads the number token from the encoded body bytes, so the signed text equals the sent text (2.5). No second formatting step anywhere |
| Form to send | UNCONFIRMED: whether the server normalises numbers before verifying. Only the minimal form (`100`, `10.5`, no trailing zeros) is proven **[existing]**. README must say: pass amounts without trailing zeros until confirmed |

Response fields whose wire type is a JSON string use `Decimal` too (it reads both), so callers see one money type.

| Where | Field | Wire type (docs) | Go |
|---|---|---|---|
| #8 `CardProduct` | `applyFee` | number (example string `'7.00'`) | `Decimal` |
| #8 | `rechargeFee` (rate 0..1) | number (example string `'0.01000000'`) | `Decimal` |
| #8 | `activeMinLimit` | number (example string `'20.00000000'`) | `Decimal` |
| #8 | `rechargeMinLimit` | number (example string `'9.00000000'`) | `Decimal` |
| #14 request | `amount` | number, max 2 decimals | `Decimal` |
| #15 request | `amount` | number, max 2 decimals | `Decimal` |
| #17 | `balance` | string | `Decimal` |
| #17 | `cardLimit` | string, optional | `Decimal` |
| #23 request | `noPinLimit` | number | `Decimal` |
| #24/#25 `Transaction` | `localCurrencyAmt` | number (example `13`) | `Decimal` |
| #24/#25 | `feeAmount` | number (example `0`) | `Decimal` |
| #24/#25 | `transCurrencyAmt` | number (example empty string `''`) | `Decimal` |
| #28 `DeliveryRegion` | `price` | string (`'48.00'`) | `Decimal` |
| #32 `MerchantAsset` | `availableAmount`, `frozenAmount` | string | `Decimal` |
| #33 request | `transactionAmount` | number | `Decimal` |
| #34 request | `transactionAmount`, `billAmount` | number | `Decimal` |
| #35 request | `transactionAmount`, `billAmount` | number | `Decimal` |
| #36 request | `transactionAmount`, `billAmount` | number | `Decimal` |
| #37 request | `transactionAmount`, `billAmount` | number | `Decimal` |
| webhook `3ds` | `transactionAmount` | string | `Decimal` |
| webhook `cardStatusChange` | `refundAmount` | string, optional | `Decimal` |
| webhook `cardRechargeResult` | `rechargeAmount` | string | `Decimal` |
| webhook `cardWithdrawResult` | `withdrawAmount`, `fee` | string | `Decimal` |
| webhook `transactionCreated` | `transaction.localCurrencyAmt`, `transaction.balance`, `transaction.feeAmount`, `transaction.transCurrencyAmt` | string | `Decimal` |
| webhook `merchantRecharge` | `amount` | **number** | `Decimal` |
| webhook `merchantTradeFee` | `amount` | string | `Decimal` |

A TypeScript SDK cannot fix the number-typed fields above at the SDK level (skill section 5): tell the user.

## 7. Webhooks

MyPal sends `POST` with a JSON body to the merchant's configured URL. The signature is in request header `sign`. All events share one URL; dispatch on `eventType`.

### 7.1 Verification (port of the Java sample in "08.Webhook / Overview")

`expected = base64std( HMAC-SHA256( key = UTF-8 bytes of webhook-aesKey, msg = UTF-8 bytes of getSortedString(rawBody) ) )`, compared with header `sign` in constant time.

The key is used **as the literal string** (`key.getBytes(UTF_8)`); it is not base64-decoded, although it looks like base64 **[verified]**.

`getSortedString(rawBody)`, rule by rule:

| # | Rule |
|---|---|
| 1 | Blank body gives `""` |
| 2 | Parse the body as a JSON object. Work on the raw tokens (`json.RawMessage` / `UseNumber`): a number contributes its literal text |
| 3 | Skip the top-level key `sign` |
| 4 | Top-level value that is a JSON object: replace by `getChildSortedString` (rule 8) |
| 5 | Any other top-level value is used as is: string gives raw text, number its text, boolean `true`/`false` |
| 6 | Sort top-level keys ascending (Java `TreeMap<String>`), join as `key=value&` |
| 7 | Remove every `"` from the result, then remove the final `&` |
| 8 | `getChildSortedString(obj)`: all keys of the object (no key is skipped), sorted ascending, joined as `key:value,`, every `"` removed, last `,` removed, wrapped in `{` `}`. An object with no keys gives `""` (not `{}`). Only this second level is canonicalised |
| 9 | Values the sample does not special-case, as the Java code would print them: `null` gives the text `null`; `""` gives `key=` (the key stays); an array or a third-level object gives its compact JSON text with quotes removed. UNCONFIRMED: none of these is shown in a worked example, and third-level key order is not defined |

**[existing]** integration A (production) differs from rule 9: it drops only keys whose value is `null`, at every level. A `""` value keeps its key (`key=`, nested `k:`), an empty object prints `{}`, arrays print `[v1,v2]` (null elements dropped) and deeper objects are recursed with sorted keys. SDK: compute the Java-sample form first; when it does not match, compute the integration A form and accept if either matches. Both need the secret key, but a body then has two valid signatures, so the fallback is a known weakening; remove it once the open question is answered.

Canonicalisation collisions: the signed string drops `"` and does not escape `&`, `=`, `,`, `:`, so one signature can cover several bodies. After a signature matches, the SDK rejects with `ErrAmbiguousWebhookBody`: (a) a body with two keys equal when compared case-insensitively, at any level (also a key holding a quote, `&`, `=`, `,`, `:`, `{` or `}`); (b) an identifier-like string value (`eventId`, `eventType`, `orderId`, `orderNo`, `status`, `txHash`, `cardNo`, `authId`, `otp`, `verificationType`, `scene`, card ids, currencies, amounts, `createAt`; also any string value inside a nested object except `merchantName`, `respCodeDesc`) that contains `"`, `&` or `=` (nested also `,` `:` `{` `}`), or is the text `null`; (c) a free-text value that imitates a `key=value` pair (`a&b=c`) or a nested pair (`a,b:c`). `ParseWebhookEvent` builds the event from the same decoded map, matching keys exactly (case-sensitive), and runs the same checks.

Test vector (Overview page, **[verified]**: recomputed, matches):

| Item | Value |
|---|---|
| Body | `{"eventType":"kycStatusChange","eventId":"03bb28da57c14e9ba500f25605ae341a","uid":10000,"auditState":1,"auditRemark":"KYC has passed"}` (the doc prints it pretty-printed; whitespace does not matter) |
| Sorted string | `auditRemark=KYC has passed&auditState=1&eventId=03bb28da57c14e9ba500f25605ae341a&eventType=kycStatusChange&uid=10000` |
| Key (`webhook-aesKey`, example printed in the docs) | `mHy3oMMoHlifsrbU5hbXQV0i5lDspm7UZosRnM/4u80=` |
| `sign` | `bYcf/qzPEa8n/uwpAIbUFvRNcgBaQo0TC5H3sjRV7bM=` |

Add a second, SDK-computed vector with a nested object (`transactionCreated`) so rule 8 is covered.

### 7.2 Receiver response and retries

| Item | Value |
|---|---|
| Success response body | `{"code":0,"msg":"Received successfully"}` (doc sample prints it as `{ "code" : 0 , "msg": "Received successfully"}`) |
| Failure response body | `{"code":1000,"msg":"fail"}` |
| Content type in the sample | `text/html; charset=UTF-8`. UNCONFIRMED: whether `application/json` is equally accepted and whether MyPal judges success by HTTP status, by `code`, or both. SDK helper writes HTTP 200 with the JSON body |
| Retries | at most 7 retries, at 1, 2, 4, 8, 15, 30 and 60 minutes after the notification is created, until a successful response is received; after that the webhook is marked failed |
| Idempotency | `eventId` is the "Unique ID for the event"; receivers must de-duplicate on it (retries re-deliver) |

### 7.3 Events

Common fields: `eventId: str`, `eventType: str`. `oldUserBankcardId?: str` ("Old system card ID") appears on every card event.

| `eventType` | Page | Fields (besides the common ones) |
|---|---|---|
| `3ds` | Authorization of 3DS Notice | `userBankcardId: id` (schema string, example integer), `verificationType: VerificationType`, `otp?: str` (when `verificationType=otp`), `authId?: str` (when `verificationType=http`; use with #26/#27), `transactionAmount: money` (string), `transactionCurrency: str`, `merchantName: str`, `scene: ThreeDSScene` (required in schema, absent in the example), `oldUserBankcardId?: str` |
| `cardStatusChange` | Bank card status change notification | `userBankcardId: id` (schema string, example integer), `cardNo: str`, `status: CardStatusEvent`, `refundAmount?: money` (string; only after card cancellation), `reason?: str` (reason for freezing), `oldUserBankcardId?: str` |
| `cardRechargeResult` | Card recharge result notification | `userBankcardId: id` (integer), `cardNo: str`, `rechargeAmount: money` (string), `orderId: str` (the merchant's `requestOrderId` of #14), `status: OrderStatus` (`SUCCESS` \| `FAILURE`; required in schema, absent in the example), `oldUserBankcardId?: str` |
| `cardWithdrawResult` | Card withdraw result notification | `userBankcardId: id` (integer), `cardNo: str`, `withdrawAmount: money` (string), `fee: money` (string), `orderId: str` (the merchant's `requestOrderId` of #15), `status: OrderStatus` (`SUCCESS` \| `FAILURE`), `oldUserBankcardId?: str`. The page's example shows `eventType: cardRechargeResult` while the description says `cardWithdrawResult`: UNCONFIRMED which value is really sent |
| `transactionCreated` | Transaction notification | `userBankcardId: id` (integer), `cardNo: str`, `createAt: int` ("tansaction time"; UNCONFIRMED: unit), `currency: str`, `transaction: WebhookTransaction`, `oldUserBankcardId?: str`. No example in the docs |
| `merchantRecharge` | Merchant recharge notification | `amount: money` (**number**), `curreny: str` (sic, spelled this way in the schema; UNCONFIRMED: real key, SDK decodes `curreny` and `currency`), `txTime: int` (UNCONFIRMED: unit), `txHash: str`. No example |
| `merchantTradeFee` | Merchant Transaction Fee Notification | `userBankcardId: id` (string), `orderNo: str`, `amount: money` (string), `currency: str`, `createAt: str` (string here, integer in `transactionCreated`), `oldUserBankcardId?: str`. No example |
| `kycStatusChange` | none (only the Overview example) | `uid: int`, `auditState: int` (example `1` with remark "KYC has passed"; UNCONFIRMED: other values), `auditRemark: str` |

`WebhookTransaction` (object `transaction` of `transactionCreated`):

| Field | Type | Notes |
|---|---|---|
| `localCurrency` | str | billing currency (the page swaps the descriptions of this field and the next) |
| `localCurrencyAmt` | money (string) | billing amount |
| `merchantName` | str | |
| `transStatus` | `TransStatus` (int) | page lists `1` processing, `2` success, `3` fail |
| `respCodeDesc` | str | remark |
| `transType` | `TransType` (int) | page lists `170`, `174`, `171`, `172`, `173` |
| `balance` | money (string) | card balance |
| `feeAmount` | money (string) | |
| `feeCurrency` | str | |
| `transactionId` | str | |
| `originalTransactionId` | str, optional | |
| `transCurrencyAmt` | money (string) | transaction amount |
| `transCurrency` | str | |
| `recordNo` | str | |
| `merchantCategoryCode` | str | MCC |

### 7.4 Go names in `webhook.go`

| Name | Purpose |
|---|---|
| `VerifyWebhookSignature(webhookKey string, body []byte, signHeader string) error` | returns `nil`, `ErrEmptyWebhookKey`, `ErrInvalidWebhookSignature` (also for an empty `sign`) or `ErrAmbiguousWebhookBody`; body is the raw request body |
| `WebhookSignHeader = "sign"` | header name constant |
| `ParseWebhookEvent(body []byte) (*WebhookEvent, error)` | decodes `eventId`/`eventType` and the typed payload with exact key names; an unknown `eventType` is not an error (payload stays nil, `Raw` kept); `ErrAmbiguousWebhookBody` as above |
| `WebhookEvent` | `EventID string`, `EventType WebhookEventType`, `Raw json.RawMessage`, and one pointer per payload: `ThreeDS *ThreeDSEvent`, `CardStatusChange *CardStatusChangeEvent`, `CardRechargeResult *CardRechargeResultEvent`, `CardWithdrawResult *CardWithdrawResultEvent`, `TransactionCreated *TransactionCreatedEvent`, `MerchantRecharge *MerchantRechargeEvent`, `MerchantTradeFee *MerchantTradeFeeEvent`, `KYCStatusChange *KYCStatusChangeEvent` |
| `WebhookEventType` and constants | `WebhookEvent3DS`, `WebhookEventCardStatusChange`, `WebhookEventCardRechargeResult`, `WebhookEventCardWithdrawResult`, `WebhookEventTransactionCreated`, `WebhookEventMerchantRecharge`, `WebhookEventMerchantTradeFee`, `WebhookEventKYCStatusChange` |
| `WebhookTransaction`, `VerificationType`, `ThreeDSScene`, `CardStatusEvent` | payload types and enums |
| `WebhookAck` / `WebhookAckSuccess()` / `WebhookAckFailure()` | `{code,msg}` response bodies of 7.2 |

The SDK ships no HTTP handler and no router: the caller reads the body, calls verify, then parse.

## 8. Sandbox

Never call production from tests: the test helper fails the test when the host of `MYPAL_BASE_URL` is the production host (checked first, before the key variables) and skips when `MYPAL_BASE_URL` or the keys are missing.

| Env var | Meaning | Needed for |
|---|---|---|
| `MYPAL_BASE_URL` | sandbox base URL (UNCONFIRMED value) | all sandbox tests |
| `MYPAL_APP_ID` | sandbox `appId` | all |
| `MYPAL_PRIVATE_KEY` | RSA private key, base64 PKCS#8 (or PEM) | all (one of the two) |
| `MYPAL_PRIVATE_KEY_FILE` | path to a file holding the same key | all (one of the two; the file is never committed) |
| `MYPAL_WEBHOOK_KEY` | `webhook-aesKey` | optional; no sandbox test needs it (webhook tests use the doc vector) |
| `MYPAL_API_AES_KEY` | `api-aesKey` for PIN encryption | optional; only a physical-card PIN test |
| `MYPAL_TEST_UID` | uid of an existing sandbox user with approved KYC | card, transaction, delivery reads; main flow |
| `MYPAL_TEST_CARD_ID` | `userBankcardId` of an active sandbox card owned by `MYPAL_TEST_UID` | card balance/info, transaction list, simulate |
| `MYPAL_TEST_PRODUCT_ID` | virtual card product id (optional; else first `VIRTUAL` from #8) | apply-card flow |

A test whose variables are missing skips itself. Test names start with `TestSandbox_`, files `*_integration_test.go`.

Safe read-only calls, one per resource group:

| Group | Call | Needs |
|---|---|---|
| Customer | `GetCustomerUID` with the email registered by the flow below (or `RegisterCustomer`, which creates data: see below) | none |
| KYC | `ListKYCCountries` (`name` empty and `name="China"`); `GetKYCStatus` | none; `MYPAL_TEST_UID` |
| Card | `ListCardProducts`; `ListCards`; `GetCardBalance`; `GetCardInfo` | none; uid; uid + card |
| Transactions | `ListTransactions` (`pageNum=1`, `pageSize=10`) | uid + card |
| Delivery | `ListDeliveryRegions` (`local=en_US`) | none |
| Merchant | `GetMerchantAssets` | none |

Main flow (create then read back): `RegisterCustomer` with `sdk-sandbox-test+<unix ms>@example.com` gives `uid`; `GetCustomerUID` with the same email returns the same `uid`; `GetKYCStatus` for that uid returns `uncommitted`.

Optional money flow (runs only when `MYPAL_TEST_UID` and `MYPAL_TEST_CARD_ID` are set): `GetCardBalance`, `RechargeCard` with a small amount and `requestOrderId = sdk-sandbox-test-<unix ms>`, poll `GetCardBalanceChangeOrder` (`type=RECHARGE`) until `SUCCESS`/`FAILURE` or timeout, `SimulateTrade`, `ListTransactions`. UNCONFIRMED: whether the sandbox merchant account is funded and whether recharge completes without manual action.

What the tests leave behind: one registered user per run (email `sdk-sandbox-test+...@example.com`; the API has no delete-user endpoint) and, when the money flow runs, one recharge order and one simulated transaction on the fixture card. Nothing is cleaned up. README must say so.

Real error case: `GetKYCStatus` with `UID = "1"` (a uid that does not belong to the merchant). Assert `errors.As` to `*APIError` with non-zero `Code`; expected `1008` ("the current user not belong you") or `1007`. UNCONFIRMED: which of the two and the HTTP status; the test asserts "non-zero code" and logs both until confirmed. Second candidate: `RegisterCustomer` with an already registered email gives `1011`.

"07.Simulate" endpoints are sandbox-only: they are used only in `TestSandbox_` tests, and the SDK refuses them on the production base URL.

## 9. Discrepancies with existing integration

| # | Topic | Docs | Existing code |
|---|---|---|---|
| D1 | Response envelope | endpoint pages show only the payload | integration A and integration B decode `{code,msg,data}`; earlier client unmarshals the body straight into the payload struct (would read zero values) and has no check of `code` |
| D2 | `uid` header on endpoints without uid | "Signature Notes" only define the canonical `uid=` | integration A and integration B send header `uid` empty; earlier client omits the header |
| D3 | Nonce | example 32 chars `A-Z0-9` | integration A sends 24 hex chars; others 32 chars `A-Z0-9` |
| D4 | Money in requests | `amount` is a JSON number | integration A uses `float64` for `amount` (recharge, withdraw); SDK must use `Decimal` |
| D5 | `userBankcardId` in Withdraw | integer | integration A sends a JSON string (works because quotes are stripped from the signed string; server evidently accepts both) |
| D6 | Apply card response | `userBankcardId`, `orderNo` | integration A also reads `cardNo` |
| D7 | Card info status key | schema `status`, example `cardStatus` | integration A reads `status` |
| D8 | Card list field types | `userBankcardId`, `productId` string in schema, integer in example | integration A decodes both forms (`flexInt`) |
| D9 | Transaction detail fields | `/bankcard/trade/info` schema lacks `transCurrencyAmt`, `transCurrency`, `merchantCategoryCode` | integration A decodes them for both list and detail; money fields typed `any` |
| D10 | `TransType` | 9 values including `165` card withdrawal | integration A has 8 constants, no `165` |
| D11 | Card status | values 0..9 | integration A defines an extra `StatusFreezing = 10` (not in docs; looks internal to that service: do not add to the SDK) |
| D12 | Merchant assets | `accountId`, `accountType` present | integration A decodes only `currency`, `availableAmount`, `frozenAmount` |
| D13 | Webhook canonical string, empty values and arrays | Java sample keeps every key (`key=` for `""`, `null` text), prints arrays as JSON text | integration A keeps `""` as `key=` (nested `k:`), drops only `null` keys at every level, prints `{}` for an empty object and arrays as `[v1,v2]` |
| D14 | Request canonical string, nested empty values | doc text: non-empty fields; Java sample skips nothing | earlier client and integration A skip `null`/`""` (also inside nested objects); integration B skips only top-level values that print as empty, and prints numbers via `%v` of `float64` (exponent form for large values) |
| D15 | Private key format | base64 PKCS#8 | services also accept PKCS#1 DER and PEM |
| D16 | Error handling | only the code table | integration A classifies codes (`1104`, `1105`, `1123`, `1124`, `429`, ambiguous outcomes); earlier client returns a plain error for HTTP >= 400 and has an unused `APIError{StatusCode, Message}` |
| D17 | Earlier client layout | n/a | earlier client has `register.go`, `kyc_status.go`, method names `Register`, `SubmitKYC`, `CheckKYCStatus`, `excute` (typo), `examples/`; the new SDK uses the names in sections 4 and 10 |
| D18 | Fixture message for code 429 | "too many requests, please try again later" | integration A fixture uses "Operation too frequent" |

## 10. Files

Non-test files of the SDK repo (52):

`gomypal/client.go`
`gomypal/request.go`
`gomypal/response.go`
`gomypal/errors.go`
`gomypal/money.go`
`gomypal/webhook.go`
`gomypal/signer/signer.go`
`gomypal/signer/signature-vectors.json`
`gomypal/customer_register.go`
`gomypal/customer_get_uid.go`
`gomypal/kyc_country_list.go`
`gomypal/kyc_file_upload.go`
`gomypal/kyc_submit.go`
`gomypal/kyc_update.go`
`gomypal/kyc_get_status.go`
`gomypal/card_product_list.go`
`gomypal/card_list.go`
`gomypal/card_apply.go`
`gomypal/card_can_activate.go`
`gomypal/card_activate.go`
`gomypal/card_set_pin.go`
`gomypal/card_recharge.go`
`gomypal/card_withdraw.go`
`gomypal/card_get_balance_change_order.go`
`gomypal/card_get_balance.go`
`gomypal/card_update_status.go`
`gomypal/card_close.go`
`gomypal/card_get_info.go`
`gomypal/card_update_email.go`
`gomypal/card_query_pin.go`
`gomypal/card_set_no_pin_limit.go`
`gomypal/transaction_list.go`
`gomypal/transaction_get.go`
`gomypal/transaction_3ds_approve.go`
`gomypal/transaction_3ds_reject.go`
`gomypal/delivery_region_list.go`
`gomypal/delivery_address_add.go`
`gomypal/delivery_address_update.go`
`gomypal/delivery_get_info.go`
`gomypal/merchant_get_assets.go`
`gomypal/simulate_3ds.go`
`gomypal/simulate_trade.go`
`gomypal/simulate_refund.go`
`gomypal/simulate_clear.go`
`gomypal/simulate_reversal.go`
`gomypal/example/quickstart/main.go`
`gomypal/README.md`
`gomypal/.env.example`
`gomypal/.gitignore`
`gomypal/.golangci.yml`
`gomypal/go.mod`
`gomypal/api-spec.md`

Placement notes: `ID` (flexible id) and `Response[T]` live in `response.go`; `Decimal` in `money.go`; `ErrorCode` constants in `errors.go`; `EncryptPIN` in `card_set_pin.go`, `DecryptPIN` in `card_query_pin.go`; shared resource types in the file named in 5.2. No `utils`, `common` or `helpers` file.

## 11. Open questions

1. UNCONFIRMED: sandbox base URL (tests take it from `MYPAL_BASE_URL`).
2. UNCONFIRMED: merchant dashboard URL; who generates the RSA key pair and where the public key is registered.
3. UNCONFIRMED: `timestamp` tolerance window (code `1004`).
4. UNCONFIRMED: `nonce` length/charset rules and replay window (code `1006`).
5. UNCONFIRMED: whether the `uid` header may be omitted (instead of sent empty) on endpoints without uid.
6. UNCONFIRMED: whether "07.Simulate" endpoints require the signing headers and `uid` (their pages declare no headers).
7. UNCONFIRMED: how `/upload/file` (multipart) is signed (assumed: header part only), maximum file size, required part filename/content type.
8. UNCONFIRMED: request canonical form of `""`, `null`, `0`, `false`, arrays, and of a query string (none exists today).
9. UNCONFIRMED: whether the server normalises JSON numbers before verifying the signature (trailing zeros, exponent); only the minimal form is proven.
10. UNCONFIRMED: HTTP status that accompanies a non-zero `code`; exact `msg` on success; whether `data` is `null`, `{}` or absent for empty responses.
11. UNCONFIRMED: `/bankcard/info` returns `status` or `cardStatus`.
12. UNCONFIRMED: wire type of ids documented as string with integer examples (`userBankcardId` in close / update email / queryPin / card list / activation response, `productId` in card list, `id` and `countryRegionId` in address update, region `id`, `accountId`).
13. UNCONFIRMED: `/bankcard/trade/details` paging (first page index, maximum `pageSize`, no total); time zone of `occurTime`; meaning of `transType` `202` seen in examples.
14. UNCONFIRMED: currency/unit of `noPinLimit` and of delivery `price`.
15. UNCONFIRMED: simulate `bankCardId` equals `userBankcardId`; how the caller obtains the `transactionId` created by Simulate trade.
16. UNCONFIRMED: webhook canonical form for `null`, `""`, arrays and third-level objects (Java sample vs integration A, D13).
17. UNCONFIRMED: webhook receiver success criterion (HTTP status, `code`, content type).
18. UNCONFIRMED: `eventType` actually sent for card withdraw (`cardWithdrawResult` vs the example's `cardRechargeResult`); real key of `curreny` in `merchantRecharge`; units of `createAt` / `txTime`; `kycStatusChange` payload and `auditState` values (event has no page).
19. UNCONFIRMED: which code (`1008` or `1007`) and HTTP status the sandbox returns for a foreign uid; whether the sandbox merchant account is funded for the recharge flow.
