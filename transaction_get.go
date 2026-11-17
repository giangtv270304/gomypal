package gomypal

import (
	"context"
	"net/http"
)

// TransType is the kind of a card transaction. MyPal examples also show values outside this list, such as 202.
type TransType int

const (
	TransTypeConsumption      TransType = 170
	TransTypeRefund           TransType = 171
	TransTypeFee              TransType = 172
	TransTypeATMWithdrawal    TransType = 173
	TransTypePreAuthorization TransType = 174
	TransTypeCardCancellation TransType = 175
	TransTypeCardApply        TransType = 140
	TransTypeCardTopUp        TransType = 150
	TransTypeCardWithdrawal   TransType = 165
)

// TransStatus is the state of a card transaction.
type TransStatus int

const (
	TransStatusInitial    TransStatus = 0
	TransStatusProcessing TransStatus = 1
	TransStatusSuccess    TransStatus = 2
	TransStatusFail       TransStatus = 3
)

// Transaction is one card transaction record.
type Transaction struct {
	// ID is the record id for GetTransaction.
	ID             int64 `json:"id"`
	UserBankcardID int64 `json:"userBankcardId"`
	// LocalCurrencyAmt and LocalCurrency are the billed amount and its currency.
	LocalCurrencyAmt Decimal `json:"localCurrencyAmt"`
	LocalCurrency    string  `json:"localCurrency"`
	RecordNo         string  `json:"recordNo"`
	// OccurTime is "2025-11-28 18:53:07" as sent; MyPal does not document the time zone.
	OccurTime             string      `json:"occurTime"`
	RespCodeDesc          string      `json:"respCodeDesc"`
	FeeAmount             Decimal     `json:"feeAmount"`
	FeeCurrency           string      `json:"feeCurrency"`
	MerchantName          string      `json:"merchantName"`
	TransType             TransType   `json:"transType"`
	TransStatus           TransStatus `json:"transStatus"`
	TransactionID         string      `json:"transactionId"`
	OriginalTransactionID string      `json:"originalTransactionId"`
	// TransCurrencyAmt and TransCurrency are the amount and currency at the merchant; they can be empty.
	TransCurrencyAmt     Decimal `json:"transCurrencyAmt"`
	TransCurrency        string  `json:"transCurrency"`
	MerchantCategoryCode string  `json:"merchantCategoryCode"`
	// AccountNum and FailureReason are not documented by MyPal and can be empty.
	AccountNum    string `json:"accountNum"`
	FailureReason string `json:"failureReason"`
}

// GetTransactionRequest is the input of GetTransaction.
type GetTransactionRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// ID is Transaction.ID.
	ID int64 `json:"id"`
}

// GetTransactionResponse is the data returned by GetTransaction.
type GetTransactionResponse = Transaction

// GetTransaction returns one transaction record.
func (c *Client) GetTransaction(ctx context.Context, req *GetTransactionRequest) (*Response[GetTransactionResponse], error) {
	const op = "GetTransaction"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"id", req.ID == 0},
	); err != nil {
		return nil, err
	}

	return call[GetTransactionResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/bankcard/trade/info",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
