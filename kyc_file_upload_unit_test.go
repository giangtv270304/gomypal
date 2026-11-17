package gomypal_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/giangtv270304/gomypal"
)

func TestUploadKYCFile_Success(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:      "/upload/file",
		uid:       testUID,
		body:      "fake-png-bytes",
		multipart: true,
		status:    http.StatusOK,
		response:  `{"code":0,"msg":"success","data":{"fileUrl":"https://files.example/front.png"}}`,
	})

	resp, err := client.UploadKYCFile(testContext(t), &gomypal.UploadKYCFileRequest{
		UID: testUID, FileName: uploadFileName, Content: strings.NewReader("fake-png-bytes"),
	})
	if err != nil {
		t.Fatalf("UploadKYCFile: %v", err)
	}

	if got, want := resp.Data.FileURL, "https://files.example/front.png"; got != want {
		t.Errorf("fileUrl = %q, want %q", got, want)
	}
}

func TestUploadKYCFile_APIError(t *testing.T) {
	t.Parallel()

	client := newFakeClient(t, fakeAPI{
		path:      "/upload/file",
		uid:       testUID,
		body:      "not-an-image",
		multipart: true,
		status:    http.StatusOK,
		response:  `{"code":1012,"msg":"image format error"}`,
	})

	_, err := client.UploadKYCFile(testContext(t), &gomypal.UploadKYCFileRequest{
		UID: testUID, FileName: uploadFileName, Content: strings.NewReader("not-an-image"),
	})

	assertAPIError(t, err, "UploadKYCFile", http.StatusOK, gomypal.CodeImageFormat)
}

func TestUploadKYCFile_MissingField(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: successEmpty}
	client := newMockClient(t, mock)

	_, err := client.UploadKYCFile(testContext(t), &gomypal.UploadKYCFileRequest{UID: testUID, FileName: "", Content: nil})
	assertMissingField(t, err, mock)

	_, err = client.UploadKYCFile(testContext(t), &gomypal.UploadKYCFileRequest{
		UID: "", FileName: uploadFileName, Content: strings.NewReader("x"),
	})
	assertMissingField(t, err, mock)
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)

	return len(p), nil
}

func TestUploadKYCFile_FileSizeIsCapped(t *testing.T) {
	t.Parallel()

	mock := &mockHTTPClient{status: http.StatusOK, body: `{"code":0,"msg":"success","data":{"fileUrl":"https://files.example/x.png"}}`}
	client := newMockClient(t, mock)

	_, err := client.UploadKYCFile(testContext(t), &gomypal.UploadKYCFileRequest{
		UID: testUID, FileName: uploadFileName, Content: io.LimitReader(zeroReader{}, gomypal.MaxUploadBytes+1),
	})
	if !errors.Is(err, gomypal.ErrFileTooLarge) || mock.calls != 0 {
		t.Errorf("one byte over the cap: error = %v calls = %d, want ErrFileTooLarge and no call", err, mock.calls)
	}

	if _, err := client.UploadKYCFile(testContext(t), &gomypal.UploadKYCFileRequest{
		UID: testUID, FileName: uploadFileName, Content: io.LimitReader(zeroReader{}, gomypal.MaxUploadBytes),
	}); err != nil || mock.calls != 1 {
		t.Errorf("exactly the cap: error = %v calls = %d, want it sent", err, mock.calls)
	}
}
