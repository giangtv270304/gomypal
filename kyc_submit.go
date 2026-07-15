package gomypal

import (
	"context"
	"net/http"
	"strconv"
)

type CertType int

const (
	CertTypeIDCard        CertType = 1
	CertTypePassport      CertType = 2
	CertTypeDriverLicense CertType = 3
)

type FileType int

const (
	FileTypeEVisa                    FileType = 7
	FileTypeVisaPhoto                FileType = 8
	FileTypePermanentResidentCard    FileType = 9
	FileTypeTemporaryResidencePermit FileType = 10
	FileTypeEmploymentPermitCard     FileType = 11
	FileTypeStudentResidencePermit   FileType = 12
	FileTypeOther                    FileType = 13
	FileTypeUtilityBill              FileType = 14
	FileTypeAddressProofDocument     FileType = 15
	FileTypeBankStatement            FileType = 16
	FileTypePropertyCertificate      FileType = 17
	FileTypeTaxDocument              FileType = 18
	FileTypeSocialSecurityCard       FileType = 19
	FileTypeHouseRentalAgreement     FileType = 20
)

// SubmitKYCRequest carries a user's KYC documents/info. FileType/FileURL
// /FaceURL are only sent when MyPal requires extra documentation for the
// user's country (see FileType's doc comment for the enum meaning);
// leave them zero/"" otherwise.
type SubmitKYCRequest struct {
	FirstName     string   `json:"firstName"`
	LastName      string   `json:"lastName"`
	IDNo          string   `json:"idNo"`
	Email         string   `json:"email"`
	NationCode    string   `json:"nationCode"`
	CertType      CertType `json:"certType"`
	IDURL         string   `json:"idUrl"`
	IDBackURL     string   `json:"idBackUrl,omitempty"`
	Birthday      string   `json:"birthday"`
	CountryCode   string   `json:"countryCode"`
	AreaCode      string   `json:"areaCode"`
	Phone         string   `json:"phone"`
	FileType      FileType `json:"fileType,omitempty"`
	FileURL       string   `json:"fileUrl,omitempty"`
	FaceURL       string   `json:"faceUrl,omitempty"`
	SelfieURL     string   `json:"selfieUrl,omitempty"`
	ReferenceID   string   `json:"referenceId,omitempty"`
	ReferenceType string   `json:"referenceType,omitempty"`
}

// SubmitKYC submits req for uid (the UID returned by Register).
// Verification is asynchronous — a nil error only confirms the
// submission was accepted, not that it was approved. Poll
// CheckKYCStatus to track the actual outcome.
//
// Example:
//
//	err := client.SubmitKYC(ctx, uid, &gomypal.SubmitKYCRequest{
//		FirstName:   "SAN",
//		LastName:    "ZHANG",
//		IDNo:        "1234567890",
//		Email:       "user@example.com",
//		NationCode:  "SGP",
//		CertType:    gomypal.CertTypeIDCard,
//		IDURL:       "https://.../id-front.jpg",
//		IDBackURL:   "https://.../id-back.jpg",
//		Birthday:    "1990-01-01",
//		CountryCode: "SGP",
//		AreaCode:    "86",
//		Phone:       "12345678",
//	})
func (c *Client) SubmitKYC(ctx context.Context, uid int64, req *SubmitKYCRequest) error {
	apiRequest := &request{
		Method:   http.MethodPost,
		Endpoint: "/user/kyc/apply",
		UID:      strconv.FormatInt(uid, 10),
		Params:   req,
	}

	_, err := c.excute(ctx, apiRequest)
	return err
}
