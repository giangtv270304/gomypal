package gomypal

import (
	"context"
	"io"
	"net/http"
)

const kycFileField = "idCard"

// UploadKYCFileRequest is the input of UploadKYCFile.
type UploadKYCFileRequest struct {
	// UID is the MyPal user id returned by RegisterCustomer, sent as the uid header. Required.
	UID string `json:"-"`
	// FileName is sent as the part filename. MyPal accepts .png, .pdf, .jpg and .jpeg (code 1012 otherwise).
	FileName string
	// Content is the file. It is read fully into memory before sending; more than MaxUploadBytes is ErrFileTooLarge.
	Content io.Reader
}

// UploadKYCFileResponse is the data returned by UploadKYCFile.
type UploadKYCFileResponse struct {
	// FileURL is the value for IDURL, IDBackURL, FileURL, FaceURL or SelfieURL in KYCInfo.
	FileURL string `json:"fileUrl"`
}

// UploadKYCFile uploads one KYC document as multipart/form-data (field "idCard").
// The signature covers the headers only; MyPal does not document multipart signing, so verify on your sandbox first.
func (c *Client) UploadKYCFile(ctx context.Context, req *UploadKYCFileRequest) (*Response[UploadKYCFileResponse], error) {
	const op = "UploadKYCFile"

	if req == nil {
		return nil, ErrNilRequest
	}

	if err := checkRequired(op,
		required{"fileName", req.FileName == ""},
		required{"content", req.Content == nil},
	); err != nil {
		return nil, err
	}

	return call[UploadKYCFileResponse](ctx, c, request{
		Op:          op,
		Method:      http.MethodPost,
		Path:        "/upload/file",
		UID:         req.UID,
		RequireUID:  true,
		SandboxOnly: false,
		Body:        nil,
		File:        &filePart{FieldName: kycFileField, FileName: req.FileName, Content: req.Content},
	})
}
