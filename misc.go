package copyleaks

import (
	"context"
	"net/http"
)

// ReleaseNotes are the release notes of the Copyleaks API.
type ReleaseNotes struct {
	// Updates are the release notes, newest first.
	Updates []ReleaseNote `json:"updates"`
}

// ReleaseNote is one release note of the Copyleaks API.
type ReleaseNote struct {
	// MsgType is the kind of change, such as "NEW".
	MsgType string `json:"msgType"`
	// Date is when the change was released, in ISO 8601 format.
	Date string `json:"date"`
	// HTMLDescription describes the change, as HTML.
	HTMLDescription string `json:"htmlDescription"`
}

// ReleaseNotes returns the release notes of the Copyleaks API. It needs no login.
func (c *Client) ReleaseNotes(ctx context.Context) (*ReleaseNotes, error) {
	var res ReleaseNotes
	if err := c.do(ctx, apiRequest{method: http.MethodGet, path: "/v3/release-logs.json"}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SupportedFileTypes lists the file types that the Plagiarism Checker can scan.
type SupportedFileTypes struct {
	// Textual are the extensions of the text files that SubmitFile accepts.
	Textual []string `json:"textual"`
	// OCR are the extensions of the image files that SubmitOCR accepts.
	OCR []string `json:"ocr"`
}

// SupportedFileTypes returns the file types that the Plagiarism Checker can scan. It
// needs no login.
//
// See https://docs.copyleaks.com/reference/actions/miscellaneous/supported-plagiarism-file-types.
func (c *Client) SupportedFileTypes(ctx context.Context) (*SupportedFileTypes, error) {
	var res SupportedFileTypes
	if err := c.do(ctx, apiRequest{method: http.MethodGet, path: "/v3/miscellaneous/supported-file-types"}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// OCRSupportedLanguages returns the language codes that SubmitOCR accepts in
// OCRSubmission.LangCode. It needs no login.
//
// See https://docs.copyleaks.com/reference/actions/miscellaneous/ocr-supported-languages.
func (c *Client) OCRSupportedLanguages(ctx context.Context) ([]string, error) {
	var res []string
	if err := c.do(ctx, apiRequest{method: http.MethodGet, path: "/v3/miscellaneous/ocr-languages-list"}, &res); err != nil {
		return nil, err
	}
	return res, nil
}
