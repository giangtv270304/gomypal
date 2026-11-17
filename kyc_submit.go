package gomypal

import (
	"context"
	"net/http"
)

// KYCStatus is the review state of a user's KYC.
type KYCStatus string

const (
	KYCStatusUncommitted KYCStatus = "uncommitted"
	KYCStatusWaiting     KYCStatus = "waiting"
	KYCStatusSuccess     KYCStatus = "success"
	KYCStatusFail        KYCStatus = "fail"
	KYCStatusWaitConfirm KYCStatus = "wait_confirm"
	KYCStatusWaitAudit   KYCStatus = "wait_audit"
)

// CertType is the kind of identity document.
type CertType int

const (
	// CertTypeIDCard and CertTypeDriverLicense need front and back photos; CertTypePassport needs the front only.
	CertTypeIDCard        CertType = 1
	CertTypePassport      CertType = 2
	CertTypeDriverLicense CertType = 3
)

// KYCFileType is the kind of extra document for countries with NeedCertificate.
type KYCFileType int

const (
	KYCFileTypeEVisa                  KYCFileType = 7
	KYCFileTypeVisaPhoto              KYCFileType = 8
	KYCFileTypePermanentResidentCard  KYCFileType = 9
	KYCFileTypeTemporaryResidence     KYCFileType = 10
	KYCFileTypeEmploymentPermit       KYCFileType = 11
	KYCFileTypeStudentResidencePermit KYCFileType = 12
	KYCFileTypeOther                  KYCFileType = 13
	KYCFileTypeUtilityBill            KYCFileType = 14
	KYCFileTypeIDWithAddress          KYCFileType = 15
	KYCFileTypeBankStatement          KYCFileType = 16
	KYCFileTypePropertyCertificate    KYCFileType = 17
	KYCFileTypeTaxDocument            KYCFileType = 18
	KYCFileTypeSocialSecurityCard     KYCFileType = 19
	KYCFileTypeRentalAgreement        KYCFileType = 20
)

// KYCInfo is the body of SubmitKYC and UpdateKYC. URL fields take the FileURL returned by UploadKYCFile.
type KYCInfo struct {
	// FirstName and LastName: English letters only.
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	IDNo      string `json:"idNo"`
	Email     string `json:"email"`
	// NationCode is the nationality, ISO 3166-1 alpha-3, for example "SGP".
	NationCode string   `json:"nationCode"`
	CertType   CertType `json:"certType"`
	// IDURL is the front photo. IDBackURL is the back photo, needed for CertTypeIDCard and CertTypeDriverLicense.
	IDURL     string `json:"idUrl"`
	IDBackURL string `json:"idBackUrl,omitempty"`
	// Birthday is YYYY-MM-DD.
	Birthday string `json:"birthday"`
	// CountryCode is the country of residence, alpha-3. AreaCode is the phone prefix without "+", for example "86".
	CountryCode string `json:"countryCode"`
	AreaCode    string `json:"areaCode"`
	Phone       string `json:"phone"`
	// FileType, FileURL and FaceURL are sent only when the country has NeedCertificate.
	FileType  KYCFileType `json:"fileType,omitempty"`
	FileURL   string      `json:"fileUrl,omitempty"`
	FaceURL   string      `json:"faceUrl,omitempty"`
	SelfieURL string      `json:"selfieUrl,omitempty"`
	// ReferenceID is the session id of a third-party face report; ReferenceType names the provider, for example "SUMSUB".
	ReferenceID   string `json:"referenceId,omitempty"`
	ReferenceType string `json:"referenceType,omitempty"`
}

func (info *KYCInfo) requiredFields() []required {
	return []required{
		{"firstName", info.FirstName == ""},
		{"lastName", info.LastName == ""},
		{"idNo", info.IDNo == ""},
		{"email", info.Email == ""},
		{"nationCode", info.NationCode == ""},
		{"certType", info.CertType == 0},
		{"idUrl", info.IDURL == ""},
		{"birthday", info.Birthday == ""},
		{"countryCode", info.CountryCode == ""},
		{"areaCode", info.AreaCode == ""},
		{"phone", info.Phone == ""},
	}
}

// SubmitKYCRequest is the input of SubmitKYC.
type SubmitKYCRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	KYCInfo
}

// SubmitKYCResponse is the data returned by SubmitKYC.
type SubmitKYCResponse struct{}

// SubmitKYC sends the user's KYC for review. The review is asynchronous: poll GetKYCStatus.
func (c *Client) SubmitKYC(ctx context.Context, req *SubmitKYCRequest) (*Response[SubmitKYCResponse], error) {
	const op = "SubmitKYC"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op, req.requiredFields()...); err != nil {
		return nil, err
	}

	return call[SubmitKYCResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/user/kyc/apply",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        req,
		File:        nil,
	})
}
