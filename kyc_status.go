package gomypal

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

type KYCStatus string

const (
	KYCStatusUncommitted KYCStatus = "uncommitted" // not submitted
	KYCStatusWaiting     KYCStatus = "waiting"      // pending review
	KYCStatusSuccess     KYCStatus = "success"      // approved
	KYCStatusFail        KYCStatus = "fail"         // rejected, see FailedReason
	KYCStatusWaitConfirm KYCStatus = "wait_confirm" // pending user confirmation
	KYCStatusWaitAudit   KYCStatus = "wait_audit"   // under review
)

// CheckKYCStatusResponse reports uid's current KYC state.
type CheckKYCStatusResponse struct {
	Status       KYCStatus `json:"status"`
	FailedReason string    `json:"failedReason"`
}

// CheckKYCStatus polls uid's KYC verification state.
//
// Example:
//
//	resp, err := client.CheckKYCStatus(ctx, uid)
//	if err != nil {
//		return err
//	}
//	if resp.Status == gomypal.KYCStatusFail {
//		log.Println("KYC rejected:", resp.FailedReason)
//	}
func (c *Client) CheckKYCStatus(ctx context.Context, uid int64) (*CheckKYCStatusResponse, error) {
	apiRequest := &request{
		Method:   http.MethodPost,
		Endpoint: "/user/kyc/status",
		UID:      strconv.FormatInt(uid, 10),
	}

	raw, err := c.excute(ctx, apiRequest)
	if err != nil {
		return nil, err
	}

	resp := new(CheckKYCStatusResponse)
	if err := json.Unmarshal(raw, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
