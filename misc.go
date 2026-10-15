package copyleaks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sort"
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

// ReleaseNotes returns the API release notes. No login needed. The endpoint is undocumented and may change.
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

// SupportedFileTypes returns the file types that the Plagiarism Checker can scan. No login needed.
// See https://docs.copyleaks.com/reference/actions/miscellaneous/supported-plagiarism-file-types.
func (c *Client) SupportedFileTypes(ctx context.Context) (*SupportedFileTypes, error) {
	var res SupportedFileTypes
	if err := c.do(ctx, apiRequest{method: http.MethodGet, path: "/v3/miscellaneous/supported-plagiarism-file-types"}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// OCRSupportedLanguages returns the codes accepted in OCRSubmission.LangCode. No login needed.
// See https://docs.copyleaks.com/reference/actions/miscellaneous/ocr-supported-languages.
func (c *Client) OCRSupportedLanguages(ctx context.Context) ([]string, error) {
	var res []string
	if err := c.do(ctx, apiRequest{method: http.MethodGet, path: "/v3/miscellaneous/ocr-languages-list"}, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// AllowedCrossLanguages lists the languages of cross-language scans (see Scanning.CrossLanguages).
type AllowedCrossLanguages struct {
	// DocumentLanguages are the languages that a scanned document can be in.
	DocumentLanguages LanguageCodes `json:"documentLanguages"`
	// ResultLanguages are the languages that the document can be compared against.
	ResultLanguages LanguageCodes `json:"resultLanguages"`
}

// LanguageCodes maps language names, such as "english", to ISO 639-1 codes, such as "en".
type LanguageCodes map[string]string

// UnmarshalJSON decodes the name-to-code object that the server sends, or a list of codes (as the API
// reference shows), mapping each code to itself.
func (l *LanguageCodes) UnmarshalJSON(b []byte) error {
	if t := bytes.TrimSpace(b); len(t) > 0 && t[0] == '[' {
		var codes []string
		if err := json.Unmarshal(t, &codes); err != nil {
			return err
		}
		m := make(LanguageCodes, len(codes))
		for _, code := range codes {
			m[code] = code
		}
		*l = m
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*l = m
	return nil
}

// Codes returns the codes, sorted and deduplicated, for use in CrossLanguage.Code.
func (l LanguageCodes) Codes() []string {
	codes := make([]string, 0, len(l))
	seen := make(map[string]bool, len(l))
	for _, code := range l {
		if !seen[code] {
			seen[code] = true
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	return codes
}

// AllowedCrossLanguages returns the languages of cross-language scans. No login needed.
// See https://docs.copyleaks.com/reference/actions/miscellaneous/supported-cross-languages.
func (c *Client) AllowedCrossLanguages(ctx context.Context) (*AllowedCrossLanguages, error) {
	var res AllowedCrossLanguages
	if err := c.do(ctx, apiRequest{method: http.MethodGet, path: "/v3/miscellaneous/allowed-cross-languages"}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SupportedAITextFileTypes lists the file types that AI Text Detection can scan.
type SupportedAITextFileTypes struct {
	// FileTypes are the file extensions, such as "pdf" and "docx".
	FileTypes []string `json:"supportedAiFileTypes"`
}

// SupportedAITextFileTypes returns the file types that AI Text Detection can scan. No login needed.
// See https://docs.copyleaks.com/reference/actions/miscellaneous/supported-ai-text-detection-file-types.
func (c *Client) SupportedAITextFileTypes(ctx context.Context) (*SupportedAITextFileTypes, error) {
	var res SupportedAITextFileTypes
	if err := c.do(ctx, apiRequest{method: http.MethodGet, path: "/v3/miscellaneous/supported-ai-text-detection-file-types"}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
