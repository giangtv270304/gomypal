package gomypal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
)

// ErrorCode is the business code in the response envelope. 0 is success; a value missing from the constants is still valid.
type ErrorCode int64

// Codes from the MyPal "Error code" page.
const (
	CodeSuccess                       ErrorCode = 0
	CodeAccessDenied                  ErrorCode = 403
	CodeTooManyRequests               ErrorCode = 429
	CodeSystemError                   ErrorCode = 500
	CodeBusinessFailed                ErrorCode = 1000
	CodeParamsInvalid                 ErrorCode = 1001
	CodeWrongAppID                    ErrorCode = 1002
	CodeTimestampFormat               ErrorCode = 1003
	CodeSignatureExpired              ErrorCode = 1004
	CodeSignatureInvalid              ErrorCode = 1005
	CodeNonceRepeated                 ErrorCode = 1006
	CodeUserNotRegistered             ErrorCode = 1007
	CodeUserNotYours                  ErrorCode = 1008
	CodeEmailFormat                   ErrorCode = 1009
	CodeWrongProductID                ErrorCode = 1010
	CodeEmailRegistered               ErrorCode = 1011
	CodeImageFormat                   ErrorCode = 1012
	CodeNoCard                        ErrorCode = 1013
	CodeUIDRequired                   ErrorCode = 1014
	CodeCardServiceError              ErrorCode = 1100
	CodeCardParameterError            ErrorCode = 1101
	CodeCardNotFound                  ErrorCode = 1102
	CodeCardholderNotFound            ErrorCode = 1103
	CodeCardNotActivated              ErrorCode = 1104
	CodeInsufficientBalance           ErrorCode = 1105
	CodeCardTransactionFailed         ErrorCode = 1106
	CodeKYCRequired                   ErrorCode = 1107
	CodeCardAlreadyBound              ErrorCode = 1108
	CodeCardPermissionDenied          ErrorCode = 1109
	CodeKYCInfoNotFound               ErrorCode = 1110
	CodeCardOperationNotSupported     ErrorCode = 1111
	CodeCardChannelNetworkError       ErrorCode = 1112
	CodeCardChannelTimeout            ErrorCode = 1113
	CodeCardUpstreamError             ErrorCode = 1114
	CodeCardUpstreamTimeout           ErrorCode = 1115
	CodeCardUpstreamUnavailable       ErrorCode = 1116
	CodeCardChannelConfigError        ErrorCode = 1117
	CodeCardChannelAuthentication     ErrorCode = 1118
	CodeCardChannelAuthorization      ErrorCode = 1119
	CodeInsufficientUSDForCard        ErrorCode = 1120
	CodeInsufficientUSDTForCard       ErrorCode = 1121
	CodeKYCLevelTooLow                ErrorCode = 1122
	CodeDuplicateCardRequest          ErrorCode = 1123
	CodeOrderNotFound                 ErrorCode = 1124
	CodeDeliveryServiceError          ErrorCode = 1200
	CodeDeliveryRegionNotFound        ErrorCode = 1201
	CodeDeliveryAddressNotFound       ErrorCode = 1202
	CodeDeliveryAddressInvalid        ErrorCode = 1203
	CodeDeliveryOrderNotFound         ErrorCode = 1204
	CodeDeliveryOrderNotYours         ErrorCode = 1205
	CodeMerchantSubAccountNotFound    ErrorCode = 1300
	CodeMerchantMainAccountNotFound   ErrorCode = 1301
	CodeMerchantAccountParameter      ErrorCode = 1302
	CodeMerchantInsufficientBalance   ErrorCode = 1303
	CodeMerchantRechargeFailed        ErrorCode = 1304
	CodeDuplicateMerchantRequest      ErrorCode = 1305
	CodeKYCApplicationFailed          ErrorCode = 1400
	CodeKYCAlreadyApproved            ErrorCode = 1401
	CodeKYCApplicationExists          ErrorCode = 1402
	CodeKYCPhoneInvalid               ErrorCode = 1403
	CodeKYCNameInvalid                ErrorCode = 1404
	CodeKYCDocumentRequired           ErrorCode = 1405
	CodeKYCStatusQueryFailed          ErrorCode = 1406
	CodeCustomerRegistrationFailed    ErrorCode = 1500
	CodeCustomerMerchantNotFound      ErrorCode = 1501
	CodeDuplicateCustomerRegistration ErrorCode = 1502
)

// APIError is returned for every response with HTTP status 400 or above, and for every response
// whose envelope code is not 0, including on HTTP 200. Read it with errors.As.
type APIError struct {
	// Op is the SDK method that made the call, for example "RechargeCard".
	Op string
	// StatusCode is the HTTP status.
	StatusCode int
	// Code is the envelope code, or 0 when the body was not the JSON envelope.
	Code ErrorCode
	// Message is the envelope msg, or the first 200 bytes of the body when the body was not the JSON envelope.
	Message string
	// RequestID is always empty: MyPal responses carry no request id.
	RequestID string
	// Body is the raw response body. It can hold card data: do not log it.
	Body []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gomypal: api error: op=%s status=%d code=%d message=%s", e.Op, e.StatusCode, e.Code, e.Message)
}

const maxRawMessageBytes = 200

func newAPIError(op string, statusCode int, raw []byte) *APIError {
	apiErr := &APIError{Op: op, StatusCode: statusCode, Code: 0, Message: string(raw[:min(len(raw), maxRawMessageBytes)]), RequestID: "", Body: raw}

	var head struct {
		Code *ErrorCode `json:"code"`
		Msg  string     `json:"msg"`
	}

	if err := json.Unmarshal(raw, &head); err == nil && head.Code != nil {
		apiErr.Code = *head.Code
		apiErr.Message = head.Msg
	}

	return apiErr
}

func matches(err error, statuses []int, codes []ErrorCode) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	return slices.Contains(statuses, apiErr.StatusCode) || slices.Contains(codes, apiErr.Code)
}

// IsNotFound reports HTTP 404 or a code that says the user, card, cardholder, KYC record, order,
// delivery region, address or order, or merchant account does not exist:
// 1007, 1102, 1103, 1110, 1124, 1201, 1202, 1204, 1300, 1301, 1501.
func IsNotFound(err error) bool {
	return matches(err, []int{http.StatusNotFound}, []ErrorCode{
		CodeUserNotRegistered, CodeCardNotFound, CodeCardholderNotFound, CodeKYCInfoNotFound, CodeOrderNotFound,
		CodeDeliveryRegionNotFound, CodeDeliveryAddressNotFound, CodeDeliveryOrderNotFound,
		CodeMerchantSubAccountNotFound, CodeMerchantMainAccountNotFound, CodeCustomerMerchantNotFound,
	})
}

// IsUnauthorized reports HTTP 401 or 403, or a code for a rejected app id, timestamp, signature, nonce
// or a uid owned by another merchant: 403, 1002, 1003, 1004, 1005, 1006, 1008.
func IsUnauthorized(err error) bool {
	return matches(err, []int{http.StatusUnauthorized, http.StatusForbidden}, []ErrorCode{
		CodeAccessDenied, CodeWrongAppID, CodeTimestampFormat, CodeSignatureExpired,
		CodeSignatureInvalid, CodeNonceRepeated, CodeUserNotYours,
	})
}

// IsConflict reports HTTP 409 or a code that says the thing already exists or the request is a duplicate:
// 1011, 1108, 1123, 1305, 1401, 1402, 1502.
func IsConflict(err error) bool {
	return matches(err, []int{http.StatusConflict}, []ErrorCode{
		CodeEmailRegistered, CodeCardAlreadyBound, CodeDuplicateCardRequest, CodeDuplicateMerchantRequest,
		CodeKYCAlreadyApproved, CodeKYCApplicationExists, CodeDuplicateCustomerRegistration,
	})
}

// IsRateLimited reports HTTP 429 or code 429.
func IsRateLimited(err error) bool {
	return matches(err, []int{http.StatusTooManyRequests}, []ErrorCode{CodeTooManyRequests})
}
