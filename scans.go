package copyleaks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// FileSubmission is the body of SubmitFile.
type FileSubmission struct {
	// Base64 is the content of the file, base64-encoded. Set either Base64 or Content.
	// To scan plain text, encode the text.
	Base64 string `json:"base64"`
	// Filename is the name of the file with its extension, up to 255 characters (required).
	Filename string `json:"filename"`
	// Properties configures the scan (required).
	Properties ScanProperties `json:"properties"`
	// Content is the content of the file, as it is. Set either Content or Base64. The SDK
	// encodes Content to base64 while it builds the request, which needs less memory
	// than encoding it yourself into Base64: about 1.4 times the file size.
	Content []byte `json:"-"`
}

// OCRSubmission is the body of SubmitOCR.
type OCRSubmission struct {
	// Base64 is the content of the image, base64-encoded. Set either Base64 or Content.
	Base64 string `json:"base64"`
	// Filename is the name of the image with its extension, up to 255 characters (required).
	Filename string `json:"filename"`
	// LangCode is the language of the text in the image (required). See
	// Client.OCRSupportedLanguages.
	LangCode string `json:"langCode"`
	// Properties configures the scan (required).
	Properties ScanProperties `json:"properties"`
	// Content is the content of the image, as it is. Set either Content or Base64. The
	// SDK encodes it to base64 while it builds the request, as FileSubmission.Content.
	Content []byte `json:"-"`
}

// URLSubmission is the body of SubmitURL.
type URLSubmission struct {
	// URL is the address of the page to scan (required).
	URL string `json:"url"`
	// Properties configures the scan (required).
	Properties ScanProperties `json:"properties"`
	// Verb is the HTTP method that Copyleaks uses to fetch URL: "GET" (the server
	// default), "POST" or "PUT".
	Verb string `json:"verb,omitempty"`
	// Headers are the headers that Copyleaks sends to fetch URL, as [name, value] pairs.
	// When set, Copyleaks sends no headers of its own. Use the Set-Cookie header for
	// cookies.
	Headers [][]string `json:"headers,omitempty"`
}

// SubmitFile starts a Plagiarism Checker scan of a file. The scan runs in the background:
// Copyleaks reports its status and results to Properties.Webhooks.Status (see
// ParseCompletedWebhook).
//
// The scan ID must be 3 to 36 characters from [a-z0-9!@$&=_()';:.,~-] and must not
// contain g, then o or 0, then gle (as in google).
//
// See https://docs.copyleaks.com/reference/actions/authenticity/submit-file.
func (c *Client) SubmitFile(ctx context.Context, token *AuthToken, scanID string, req *FileSubmission) error {
	if err := validateSubmitScanID(scanID); err != nil {
		return err
	}
	if req == nil || req.Filename == "" {
		return invalid("Filename and either Base64 or Content are required")
	}
	if err := checkUploadContent(req.Base64, req.Content); err != nil {
		return err
	}
	if err := req.Properties.validate(); err != nil {
		return err
	}
	if err := VerifyAuthToken(token); err != nil {
		return err
	}
	head := *req
	head.Base64, head.Content = "", nil
	body, err := base64Body(&head, req.Base64, req.Content)
	if err != nil {
		return fmt.Errorf("copyleaks: encode PUT /v3/scans/submit/file request: %w", err)
	}
	return c.do(ctx, apiRequest{
		method:      http.MethodPut,
		path:        "/v3/scans/submit/file/" + segment(scanID),
		token:       token,
		rawBody:     body,
		contentType: "application/json",
		idempotent:  true,
	}, nil)
}

// SubmitOCR starts a Plagiarism Checker scan of the text in an image. It works like
// SubmitFile.
//
// See https://docs.copyleaks.com/reference/actions/authenticity/submit-ocr.
func (c *Client) SubmitOCR(ctx context.Context, token *AuthToken, scanID string, req *OCRSubmission) error {
	if err := validateSubmitScanID(scanID); err != nil {
		return err
	}
	if req == nil || req.Filename == "" || req.LangCode == "" {
		return invalid("Filename, LangCode and either Base64 or Content are required")
	}
	if err := checkUploadContent(req.Base64, req.Content); err != nil {
		return err
	}
	if err := req.Properties.validate(); err != nil {
		return err
	}
	if err := VerifyAuthToken(token); err != nil {
		return err
	}
	head := *req
	head.Base64, head.Content = "", nil
	body, err := base64Body(&head, req.Base64, req.Content)
	if err != nil {
		return fmt.Errorf("copyleaks: encode PUT /v3/scans/submit/ocr request: %w", err)
	}
	return c.do(ctx, apiRequest{
		method:      http.MethodPut,
		path:        "/v3/scans/submit/ocr/" + segment(scanID),
		token:       token,
		rawBody:     body,
		contentType: "application/json",
		idempotent:  true,
	}, nil)
}

// SubmitURL starts a Plagiarism Checker scan of a web page. It works like SubmitFile.
//
// See https://docs.copyleaks.com/reference/actions/authenticity/submit-url.
func (c *Client) SubmitURL(ctx context.Context, token *AuthToken, scanID string, req *URLSubmission) error {
	if err := validateSubmitScanID(scanID); err != nil {
		return err
	}
	if req == nil || req.URL == "" {
		return invalid("URL is required")
	}
	if err := validateVerb(req.Verb); err != nil {
		return err
	}
	if err := req.Properties.validate(); err != nil {
		return err
	}
	if err := VerifyAuthToken(token); err != nil {
		return err
	}
	return c.do(ctx, apiRequest{
		method:     http.MethodPut,
		path:       "/v3/scans/submit/url/" + segment(scanID),
		token:      token,
		body:       req,
		idempotent: true,
	}, nil)
}

// ExportRequest is the body of Export. It lists the artifacts of a scan to upload to
// your servers.
type ExportRequest struct {
	// CompletionWebhook is called when the export is done (required); see
	// ParseExportCompletedWebhook.
	CompletionWebhook string `json:"completionWebhook"`
	// CompletionWebhookHeaders are extra headers of the CompletionWebhook request, as
	// [name, value] pairs.
	CompletionWebhookHeaders [][]string `json:"completionWebhookHeaders,omitempty"`
	// MaxRetries is how many times each upload is retried, from 1 to 12. The server
	// default is 3.
	MaxRetries int `json:"maxRetries,omitempty"`
	// DeveloperPayload is sent back in the export-completed webhook.
	DeveloperPayload string `json:"developerPayload,omitempty"`
	// Results are the results to export, by result ID from the completed webhook; see
	// ParseExportedResult.
	Results []ExportResult `json:"results,omitempty"`
	// PDFReport exports the PDF report. It requires ScanProperties.PDF.Create on submit.
	PDFReport *ExportTarget `json:"pdfReport,omitempty"`
	// CrawledVersion exports the crawled version of the submitted content; see
	// ParseExportedCrawledVersion.
	CrawledVersion *ExportTarget `json:"crawledVersion,omitempty"`
	// AIDetection exports the AI Text Detection report, with AI Logic when it was on. It
	// requires ScanProperties.AIGeneratedText.Detect on submit; see
	// ParseExportedAIDetection.
	AIDetection *ExportTarget `json:"aiDetection,omitempty"`
	// WritingFeedback exports the Grammar Checker report. It requires
	// ScanProperties.WritingFeedback.Enable on submit; see ParseExportedWritingFeedback.
	WritingFeedback *ExportTarget `json:"writingFeedback,omitempty"`
	// Overview exports the Gen-AI Overview. It requires ScanProperties.Overview.Enable on
	// submit; see ParseExportedOverview.
	Overview *ExportTarget `json:"overview,omitempty"`
}

// ExportResult uploads one result of a scan.
type ExportResult struct {
	// ID is the result ID, from the completed webhook (required).
	ID string `json:"id"`
	// Endpoint is the URL to upload the result to (required).
	Endpoint string `json:"endpoint"`
	// Verb is the HTTP method of the upload, such as "POST" (required).
	Verb string `json:"verb"`
	// Headers are extra headers of the upload, as [name, value] pairs.
	Headers [][]string `json:"headers,omitempty"`
}

// ExportTarget is where to upload one artifact of a scan.
type ExportTarget struct {
	// Endpoint is the URL to upload the artifact to (required).
	Endpoint string `json:"endpoint"`
	// Verb is the HTTP method of the upload, such as "POST" (required).
	Verb string `json:"verb"`
	// Headers are extra headers of the upload, as [name, value] pairs.
	Headers [][]string `json:"headers,omitempty"`
}

// Export uploads the artifacts of a completed scan to your servers, then calls
// req.CompletionWebhook. exportID is a new ID for the export. Both IDs must be 3 to 36
// characters from [a-z0-9!@$^&+%=_(){}<>'";:/.,~`|\-] and must not contain g, then o or
// 0, then gle (as in google).
//
// See https://docs.copyleaks.com/reference/actions/downloads/export.
func (c *Client) Export(ctx context.Context, token *AuthToken, scanID, exportID string, req *ExportRequest) error {
	if err := validateExportID("scan ID", scanID); err != nil {
		return err
	}
	if err := validateExportID("export ID", exportID); err != nil {
		return err
	}
	if err := req.validate(); err != nil {
		return err
	}
	if err := VerifyAuthToken(token); err != nil {
		return err
	}
	return c.do(ctx, apiRequest{
		method:     http.MethodPost,
		path:       "/v3/downloads/" + segment(scanID) + "/export/" + segment(exportID),
		token:      token,
		body:       req,
		idempotent: true,
		// After a 503, the export ID belongs to a failed export: a retry would only
		// get 409 Conflict.
		noRetryOn503: true,
	}, nil)
}

// validate checks the fields that the server requires.
func (r *ExportRequest) validate() error {
	if r == nil || r.CompletionWebhook == "" {
		return invalid("CompletionWebhook is required")
	}
	if r.MaxRetries != 0 && (r.MaxRetries < 1 || r.MaxRetries > 12) {
		return invalid("MaxRetries must be 1 to 12")
	}
	for _, res := range r.Results {
		if res.ID == "" || res.Endpoint == "" || res.Verb == "" {
			return invalid("each of Results needs ID, Endpoint and Verb")
		}
	}
	for _, t := range []*ExportTarget{r.PDFReport, r.CrawledVersion, r.AIDetection, r.WritingFeedback, r.Overview} {
		if t != nil && (t.Endpoint == "" || t.Verb == "") {
			return invalid("PDFReport, CrawledVersion, AIDetection, WritingFeedback and Overview need Endpoint and Verb")
		}
	}
	return nil
}

// StartRequest is the body of Start.
type StartRequest struct {
	// Trigger lists the IDs of up to 100 scans, submitted with SubmitActionCheckCredits or
	// SubmitActionIndex, to start (required).
	Trigger []string `json:"trigger"`
	// ErrorHandling is what happens when some of the scans are in error. The default,
	// StartErrorHandlingCancel, cancels the request.
	ErrorHandling StartErrorHandling `json:"errorHandling"`
}

// StartErrorHandling is what Start does when some of the scans are in error.
type StartErrorHandling int

const (
	// StartErrorHandlingCancel cancels the request. It is the default.
	StartErrorHandlingCancel StartErrorHandling = 0
	// StartErrorHandlingIgnore starts the scans that are not in error.
	StartErrorHandlingIgnore StartErrorHandling = 1
)

// StartResponse lists the scans that Start started and those it could not start.
type StartResponse struct {
	// Success are the scans that started.
	Success []StartedScan `json:"success"`
	// Failed are the scans that did not start.
	Failed []StartedScan `json:"failed"`
}

// StartedScan is one scan of a StartResponse.
type StartedScan struct {
	// ID is the scan ID.
	ID string `json:"id"`
}

// UnmarshalJSON accepts a scan as an object with an id, which the server sends, or as a
// bare scan ID string, as the API reference shows it.
func (s *StartedScan) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, &s.ID)
	}
	type plain StartedScan
	return json.Unmarshal(b, (*plain)(s))
}

// Start starts scans that were submitted with SubmitActionCheckCredits or
// SubmitActionIndex.
//
// See https://docs.copyleaks.com/reference/actions/authenticity/start.
func (c *Client) Start(ctx context.Context, token *AuthToken, req *StartRequest) (*StartResponse, error) {
	if req == nil || len(req.Trigger) == 0 || len(req.Trigger) > 100 {
		return nil, invalid("Trigger must list 1 to 100 scan IDs")
	}
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	var res StartResponse
	err := c.do(ctx, apiRequest{
		method: http.MethodPatch,
		path:   "/v3/scans/start",
		token:  token,
		body:   req,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// DeleteRequest is the body of Delete.
type DeleteRequest struct {
	// Scans are the completed scans to delete, up to 10,000 (required).
	Scans []DeleteScan `json:"scans"`
	// Purge also removes every trace of the scans from the Shared Data Hub and the Private
	// Cloud Hubs, so they are never found as results again.
	Purge bool `json:"purge,omitempty"`
	// CompletionWebhook is called with a POST request, without a body, when the deletion
	// is done.
	CompletionWebhook string `json:"completionWebhook,omitempty"`
	// Headers are extra headers of the CompletionWebhook request, as [name, value] pairs.
	Headers [][]string `json:"headers,omitempty"`
}

// DeleteScan is one scan to delete.
type DeleteScan struct {
	// ID is the scan ID.
	ID string `json:"id"`
}

// Delete deletes completed scans, with their results and metadata. The deletion runs in
// the background and can take a few minutes.
//
// See https://docs.copyleaks.com/reference/actions/authenticity/delete.
func (c *Client) Delete(ctx context.Context, token *AuthToken, req *DeleteRequest) error {
	if req == nil || len(req.Scans) == 0 || len(req.Scans) > 10000 {
		return invalid("Scans must list 1 to 10000 scans")
	}
	for _, s := range req.Scans {
		if s.ID == "" {
			return invalid("each of Scans needs an ID")
		}
	}
	if err := VerifyAuthToken(token); err != nil {
		return err
	}
	return c.do(ctx, apiRequest{
		method: http.MethodPatch,
		path:   "/v3.1/scans/delete",
		token:  token,
		body:   req,
	}, nil)
}

// ResendWebhook asks Copyleaks to send the status webhook of a finished scan again.
//
// See https://docs.copyleaks.com/reference/actions/authenticity/resend-webhook.
func (c *Client) ResendWebhook(ctx context.Context, token *AuthToken, scanID string) error {
	if err := validateID("scan ID", scanID); err != nil {
		return err
	}
	if err := VerifyAuthToken(token); err != nil {
		return err
	}
	return c.do(ctx, apiRequest{
		method: http.MethodPost,
		path:   "/v3/scans/" + segment(scanID) + "/webhooks/resend",
		token:  token,
	}, nil)
}

// CreditsBalance is the credit balance of an account.
type CreditsBalance struct {
	// Amount is the number of remaining credits. A credit covers up to 250 words.
	Amount int `json:"Amount"`
}

// CreditsBalance returns the credit balance of the account.
//
// See https://docs.copyleaks.com/reference/actions/admin/check-credits.
func (c *Client) CreditsBalance(ctx context.Context, token *AuthToken) (*CreditsBalance, error) {
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	var res CreditsBalance
	err := c.do(ctx, apiRequest{
		method: http.MethodGet,
		path:   "/v3/scans/credits",
		token:  token,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// usageDateLayout is the dd-MM-yyyy date format of UsagesHistoryCSV.
const usageDateLayout = "02-01-2006"

// UsagesHistoryCSV returns the usage history of the account between two dates,
// inclusive, as a CSV file. Only the dates of start and end are used.
//
// See https://docs.copyleaks.com/reference/actions/admin/usage-history.
func (c *Client) UsagesHistoryCSV(ctx context.Context, token *AuthToken, start, end time.Time) ([]byte, error) {
	if start.IsZero() || end.IsZero() {
		return nil, invalid("start and end dates are required")
	}
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	var csv []byte
	err := c.do(ctx, apiRequest{
		method: http.MethodGet,
		path:   "/v3/scans/usages/history",
		query:  url.Values{"start": {start.Format(usageDateLayout)}, "end": {end.Format(usageDateLayout)}},
		token:  token,
	}, &csv)
	if err != nil {
		return nil, err
	}
	return csv, nil
}
