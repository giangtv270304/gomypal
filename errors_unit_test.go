package gomypal_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestErrorHelpers(t *testing.T) {
	t.Parallel()

	type helper struct {
		name  string
		check func(error) bool
	}

	notFound := helper{"IsNotFound", gomypal.IsNotFound}
	unauthorized := helper{"IsUnauthorized", gomypal.IsUnauthorized}
	conflict := helper{"IsConflict", gomypal.IsConflict}
	rateLimited := helper{"IsRateLimited", gomypal.IsRateLimited}
	helpers := []helper{notFound, unauthorized, conflict, rateLimited}

	testCases := []struct {
		status int
		code   gomypal.ErrorCode
		want   string
	}{
		{http.StatusNotFound, 0, notFound.name},
		{http.StatusUnauthorized, 0, unauthorized.name},
		{http.StatusForbidden, 0, unauthorized.name},
		{http.StatusConflict, 0, conflict.name},
		{http.StatusTooManyRequests, 0, rateLimited.name},
		{http.StatusOK, gomypal.CodeTooManyRequests, rateLimited.name},
		{http.StatusOK, gomypal.CodeAccessDenied, unauthorized.name},
		{http.StatusOK, gomypal.CodeWrongAppID, unauthorized.name},
		{http.StatusOK, gomypal.CodeTimestampFormat, unauthorized.name},
		{http.StatusOK, gomypal.CodeSignatureExpired, unauthorized.name},
		{http.StatusOK, gomypal.CodeSignatureInvalid, unauthorized.name},
		{http.StatusOK, gomypal.CodeNonceRepeated, unauthorized.name},
		{http.StatusOK, gomypal.CodeUserNotYours, unauthorized.name},
		{http.StatusOK, gomypal.CodeUserNotRegistered, notFound.name},
		{http.StatusOK, gomypal.CodeCardNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeCardholderNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeKYCInfoNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeOrderNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeDeliveryRegionNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeDeliveryAddressNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeDeliveryOrderNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeMerchantSubAccountNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeMerchantMainAccountNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeCustomerMerchantNotFound, notFound.name},
		{http.StatusOK, gomypal.CodeEmailRegistered, conflict.name},
		{http.StatusOK, gomypal.CodeCardAlreadyBound, conflict.name},
		{http.StatusOK, gomypal.CodeDuplicateCardRequest, conflict.name},
		{http.StatusOK, gomypal.CodeDuplicateMerchantRequest, conflict.name},
		{http.StatusOK, gomypal.CodeKYCAlreadyApproved, conflict.name},
		{http.StatusOK, gomypal.CodeKYCApplicationExists, conflict.name},
		{http.StatusOK, gomypal.CodeDuplicateCustomerRegistration, conflict.name},
		{http.StatusOK, gomypal.CodeInsufficientBalance, ""},
		{http.StatusOK, 987654, ""},
		{http.StatusInternalServerError, gomypal.CodeSystemError, ""},
	}

	for _, testCase := range testCases {
		apiErr := &gomypal.APIError{Op: "X", StatusCode: testCase.status, Code: testCase.code, Message: "m", RequestID: "", Body: nil}
		wrapped := fmt.Errorf("outer: %w", apiErr)

		for _, candidate := range helpers {
			if got, want := candidate.check(wrapped), candidate.name == testCase.want; got != want {
				t.Errorf("status %d code %d: %s = %v, want %v", testCase.status, testCase.code, candidate.name, got, want)
			}
		}
	}

	for _, candidate := range helpers {
		if candidate.check(errors.New("plain")) || candidate.check(nil) {
			t.Errorf("%s is true for a non-API error", candidate.name)
		}
	}
}
