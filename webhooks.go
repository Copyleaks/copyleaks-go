package copyleaks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// AlertCodeSuspectedAIText is the code of the alert that carries the AI Text Detection result of a scan with
// ScanProperties.AIGeneratedText.
const AlertCodeSuspectedAIText = "suspected-ai-text"

// CompletedWebhook is sent to the status webhook when a Plagiarism Checker scan succeeds.
// See https://docs.copyleaks.com/reference/data-types/authenticity/webhooks/scan-completed.
type CompletedWebhook struct {
	// Status is the scan status: 0 for success.
	Status int `json:"status"`
	// DeveloperPayload is the ScanProperties.DeveloperPayload of the scan.
	DeveloperPayload string `json:"developerPayload"`
	// ScannedDocument is general information about the scanned document.
	ScannedDocument *WebhookScannedDocument `json:"scannedDocument"`
	// Results are the results that were found, by source, and the aggregated score.
	Results *Results `json:"results"`
	// Notifications are the scan alerts.
	Notifications *Notifications `json:"notifications"`
	// ReferencesValidation is the Citation Checker summary; per-reference results are in
	// ExportedCrawledVersion.ReferencesValidation.
	ReferencesValidation *ReferencesValidation `json:"referencesValidation"`
	// WritingFeedback is the Grammar Checker summary; export the full report with ExportRequest.WritingFeedback.
	WritingFeedback *WebhookWritingFeedback `json:"writingFeedback"`
}

// WebhookWritingFeedback is the Grammar Checker summary of a completed scan.
type WebhookWritingFeedback struct {
	// TextStatistics are statistics of the text.
	TextStatistics *GrammarStatistics `json:"textStatistics"`
	// Score holds the correction counts and scores of each category, and the overall score.
	Score *GrammarScoreCorrections `json:"score"`
	// Readability is the readability of the text.
	Readability *GrammarReadability `json:"readability"`
}

// ReferencesValidation is the result of the Citation Checker.
type ReferencesValidation struct {
	// Summary counts the references that were found and validated.
	Summary *ReferencesValidationSummary `json:"summary"`
	// Results are the validated references; only the crawled version has them, not the completed webhook.
	Results []ReferenceValidation `json:"results,omitempty"`
}

// ReferencesValidationSummary counts the references that the Citation Checker found and validated.
type ReferencesValidationSummary struct {
	// Total is the number of references found.
	Total int `json:"total"`
	// Academic is the number of references found to be academic.
	Academic int `json:"academic"`
	// AcademicValidated is the number of academic references that were fully corroborated.
	AcademicValidated int `json:"academicValidated"`
	// NonAcademicValidated is the number of other references fully corroborated by their cited source.
	NonAcademicValidated int `json:"nonAcademicValidated"`
}

// ErrorWebhook is the payload of the error webhook, sent when a scan ends with an error.
// See https://docs.copyleaks.com/reference/data-types/authenticity/webhooks/error.
type ErrorWebhook struct {
	// Status is the scan status: 1 for error.
	Status int `json:"status"`
	// DeveloperPayload is the ScanProperties.DeveloperPayload of the scan.
	DeveloperPayload string `json:"developerPayload"`
	// Error describes the error.
	Error *WebhookError `json:"error"`
}

// WebhookError describes why a scan failed.
type WebhookError struct {
	// Type is the error type, such as "invalid_request_error".
	Type string `json:"type"`
	// ID is the machine-readable error ID, such as "insufficient_credits".
	ID string `json:"id"`
	// Code is the numeric error code.
	Code int `json:"code"`
	// Message is a human-readable description of the error.
	Message string `json:"message"`
	// URL links to the documentation of the error.
	URL string `json:"url"`
	// Details describe the request parameters that caused the error.
	Details []WebhookErrorDetail `json:"details"`
}

// WebhookErrorDetail describes one request parameter that caused an error.
type WebhookErrorDetail struct {
	// Param is the name of the parameter.
	Param string `json:"param"`
	// Message describes the problem with the parameter.
	Message string `json:"message"`
}

// CreditsCheckedWebhook is the creditsChecked webhook payload of a scan submitted with SubmitActionCheckCredits.
// See https://docs.copyleaks.com/reference/data-types/authenticity/webhooks/credits-checked.
type CreditsCheckedWebhook struct {
	// Status is the scan status: 2 for credits checked.
	Status int `json:"status"`
	// DeveloperPayload is the ScanProperties.DeveloperPayload of the scan.
	DeveloperPayload string `json:"developerPayload"`
	// Credits is the cost of the scan, if you start it with Client.Start.
	Credits int `json:"credits"`
	// ScannedDocument is general information about the scanned document.
	ScannedDocument *WebhookScannedDocument `json:"scannedDocument"`
}

// IndexedWebhook is the indexed webhook payload of a scan submitted with SubmitActionIndex.
// See https://docs.copyleaks.com/reference/data-types/authenticity/webhooks/indexed.
type IndexedWebhook struct {
	// Status is the scan status: 3 for indexed.
	Status int `json:"status"`
	// DeveloperPayload is the ScanProperties.DeveloperPayload of the scan.
	DeveloperPayload string `json:"developerPayload"`
}

// NewResultWebhook is sent to SubmitWebhooks.NewResult each time a result is found while the scan runs.
// See https://docs.copyleaks.com/reference/data-types/authenticity/results/new-result.
type NewResultWebhook struct {
	// DeveloperPayload is the ScanProperties.DeveloperPayload of the scan.
	DeveloperPayload string `json:"developerPayload"`
	// Score is the score of the scan so far.
	Score *ResultsScore `json:"score"`
	// Internet are the new results from the internet.
	Internet []InternetResult `json:"internet"`
	// Database are the new results from the Shared Data Hub.
	Database []DatabaseResult `json:"database"`
	// Batch are the new results from the same batch of scans.
	Batch []DatabaseResult `json:"batch"`
	// Repositories are the new results from Private Cloud Hubs.
	Repositories []RepositoryResult `json:"repositories"`
	// InternalAIData are the new results from the Copyleaks internal AI database.
	InternalAIData []InternalAIResult `json:"internalAIData"`
}

// ExportCompletedWebhook is sent to ExportRequest.CompletionWebhook when an export is done. Check that Completed
// is true and that every task is healthy with a 2xx HTTPStatusCode.
// See https://docs.copyleaks.com/reference/data-types/authenticity/webhooks/export-completed.
type ExportCompletedWebhook struct {
	// Completed reports whether the export finished without internal errors.
	Completed bool `json:"completed"`
	// DeveloperPayload is the ExportRequest.DeveloperPayload of the export.
	DeveloperPayload string `json:"developerPayload"`
	// Tasks are the uploads of the export.
	Tasks []ExportTask `json:"tasks"`
}

// ExportTask is one upload of an export.
type ExportTask struct {
	// Endpoint is the URL that the artifact was uploaded to.
	Endpoint string `json:"endpoint"`
	// IsHealthy reports whether the upload succeeded.
	IsHealthy bool `json:"isHealthy"`
	// HTTPStatusCode is the status code that your server answered with.
	HTTPStatusCode int `json:"httpStatusCode"`
}

// WebhookScannedDocument is general information about a scanned document.
type WebhookScannedDocument struct {
	// ScanID is the scan ID given by you.
	ScanID string `json:"scanId"`
	// TotalWords is the number of words in the document.
	TotalWords int `json:"totalWords"`
	// TotalExcluded is the number of excluded words.
	TotalExcluded int `json:"totalExcluded"`
	// Credits is the number of credits that the scan used.
	Credits int `json:"credits"`
	// CreationTime is when the scan was created, in ISO 8601 format (UTC).
	CreationTime string `json:"creationTime"`
	// Metadata is metadata of the scanned document.
	Metadata *ResultMetadata `json:"metadata"`
	// ExpectedCredits is the number of credits that the scan was expected to use.
	ExpectedCredits int `json:"expectedCredits"`
	// DetectedLanguage is the language code of the document, such as "en", when the server reports it.
	DetectedLanguage string `json:"detectedLanguage"`
	// Enabled reports the features that the scan ran, when the server reports them.
	Enabled *ScanEnabledFeatures `json:"enabled"`
}

// ScanEnabledFeatures reports the features that a scan ran.
type ScanEnabledFeatures struct {
	// PlagiarismDetection reports the Plagiarism Checker.
	PlagiarismDetection bool `json:"plagiarismDetection"`
	// AIDetection reports AI Text Detection.
	AIDetection bool `json:"aiDetection"`
	// ExplainableAI reports AI Logic.
	ExplainableAI bool `json:"explainableAi"`
	// WritingFeedback reports the Grammar Checker.
	WritingFeedback bool `json:"writingFeedback"`
	// PDFReport reports the PDF report.
	PDFReport bool `json:"pdfReport"`
	// CheatDetection reports cheat detection.
	CheatDetection bool `json:"cheatDetection"`
	// Overview reports the Gen-AI Overview.
	Overview bool `json:"overview"`
	// ReferencesValidation reports the Citation Checker.
	ReferencesValidation bool `json:"referencesValidation"`
	// AISourceMatch reports AI Source Match.
	AISourceMatch bool `json:"aiSourceMatch"`
	// InternalAISourceMatch reports Internal AI Source Match.
	InternalAISourceMatch bool `json:"internalAiSourceMatch"`
}

// Results are the results of a scan, by source.
type Results struct {
	// Internet are the results from the internet.
	Internet []InternetResult `json:"internet"`
	// Database are the results from the Shared Data Hub.
	Database []DatabaseResult `json:"database"`
	// Batch are the results from the same batch of scans.
	Batch []DatabaseResult `json:"batch"`
	// Repositories are the results from Private Cloud Hubs.
	Repositories []RepositoryResult `json:"repositories"`
	// Score is the aggregated score of all results.
	Score *ResultsScore `json:"score"`
	// InternalAIData are the results from the Copyleaks internal AI database.
	InternalAIData []InternalAIResult `json:"internalAIData"`
}

// InternalAIResult is a result found in the Copyleaks internal AI database.
type InternalAIResult struct {
	// ID is the result ID.
	ID string `json:"id"`
	// Title is the title of the matching entry.
	Title string `json:"title"`
	// Introduction is an excerpt of the matching content.
	Introduction string `json:"introduction"`
	// MatchedWords is the number of words that match the scanned document.
	MatchedWords int `json:"matchedWords"`
	// IdenticalWords is the number of identical words.
	IdenticalWords int `json:"identicalWords"`
	// SimilarWords is the number of similar, but not identical, words.
	SimilarWords int `json:"similarWords"`
	// ParaphrasedWords is the number of paraphrased words.
	ParaphrasedWords int `json:"paraphrasedWords"`
	// TotalWords is the number of words in the matching entry.
	TotalWords int `json:"totalWords"`
	// Tags are the tags of the result.
	Tags []ResultTag `json:"tags"`
}

// InternetResult is a result found on the internet.
type InternetResult struct {
	// ID is the result ID, used by Client.Export.
	ID string `json:"id"`
	// Title is the title of the source.
	Title string `json:"title"`
	// Introduction is a short introduction of the source.
	Introduction string `json:"introduction"`
	// MatchedWords is the number of words that match the scanned document.
	MatchedWords int `json:"matchedWords"`
	// URL is the public URL of the source.
	URL string `json:"url"`
	// Metadata is metadata of the source.
	Metadata *ResultMetadata `json:"metadata"`
	// Tags are the tags of the result.
	Tags []ResultTag `json:"tags"`
}

// DatabaseResult is a result found in the Shared Data Hub or in the same batch of scans.
type DatabaseResult struct {
	// ID is the result ID, used by Client.Export.
	ID string `json:"id"`
	// Title is the title of the source.
	Title string `json:"title"`
	// Introduction is a short introduction of the source.
	Introduction string `json:"introduction"`
	// MatchedWords is the number of words that match the scanned document.
	MatchedWords int `json:"matchedWords"`
	// ScanID is the scan ID of the source, when you submitted it.
	ScanID string `json:"scanId"`
	// Metadata is metadata of the source.
	Metadata *ResultMetadata `json:"metadata"`
	// Tags are the tags of the result.
	Tags []ResultTag `json:"tags"`
}

// RepositoryResult is a result found in a Private Cloud Hub.
type RepositoryResult struct {
	// ID is the result ID, used by Client.Export.
	ID string `json:"id"`
	// Title is the title of the source.
	Title string `json:"title"`
	// Introduction is a short introduction of the source.
	Introduction string `json:"introduction"`
	// MatchedWords is the number of words that match the scanned document.
	MatchedWords int `json:"matchedWords"`
	// RepositoryID is the ID of the Private Cloud Hub.
	RepositoryID string `json:"repositoryId"`
	// ScanID is the scan ID of the source in the Private Cloud Hub.
	ScanID string `json:"scanId"`
	// Metadata is metadata of the source.
	Metadata *ResultMetadata `json:"metadata"`
	// Tags are the tags of the result.
	Tags []ResultTag `json:"tags"`
}

// ResultMetadata is metadata of a scanned document or of a result.
type ResultMetadata struct {
	// FinalURL is the URL after all HTTP redirects.
	FinalURL string `json:"finalUrl"`
	// CanonicalURL is the canonical URL of the page.
	CanonicalURL string `json:"canonicalUrl"`
	// PublishDate is the publication date.
	PublishDate string `json:"publishDate"`
	// CreationDate is the creation date.
	CreationDate string `json:"creationDate"`
	// LastModificationDate is the last modification date.
	LastModificationDate string `json:"lastModificationDate"`
	// Author is the author.
	Author string `json:"author"`
	// Organization is the organization.
	Organization string `json:"organization"`
	// Filename is the file name.
	Filename string `json:"filename"`
	// SubmittedBy is the email of the user who added the document; only Private Cloud Hub results have it.
	SubmittedBy string `json:"submittedBy"`
}

// ResultTag is a tag of a result.
type ResultTag struct {
	// Code identifies the kind of tag.
	Code string `json:"code"`
	// Title is a short name of the tag.
	Title string `json:"title"`
	// Description describes the tag.
	Description string `json:"description"`
}

// ResultsScore is the aggregated score of the results of a scan.
type ResultsScore struct {
	// IdenticalWords is the number of words that match exactly.
	IdenticalWords int `json:"identicalWords"`
	// MinorChangedWords is the number of nearly identical words, such as slow and slowly.
	MinorChangedWords int `json:"minorChangedWords"`
	// RelatedMeaningWords is the number of paraphrased words.
	RelatedMeaningWords int `json:"relatedMeaningWords"`
	// AggregatedScore is the percentage of similar words from all results.
	AggregatedScore float64 `json:"aggregatedScore"`
}

// UnmarshalJSON decodes a score object (as the server sends) or a bare number as AggregatedScore (as the API
// reference shows for the new-result webhook).
func (s *ResultsScore) UnmarshalJSON(b []byte) error {
	if t := bytes.TrimSpace(b); len(t) > 0 && t[0] != '{' && t[0] != 'n' {
		*s = ResultsScore{}
		return json.Unmarshal(t, &s.AggregatedScore)
	}
	type plain ResultsScore
	return json.Unmarshal(b, (*plain)(s))
}

// Notifications are the scan alerts of a completed scan.
type Notifications struct {
	// Alerts are the scan alerts.
	Alerts []Alert `json:"alerts"`
}

// Alert is a scan alert.
// See https://docs.copyleaks.com/reference/data-types/authenticity/scan-alerts.
type Alert struct {
	// Category is the alert category; 2 is AI Text Detection.
	Category int `json:"category"`
	// Code identifies the alert, such as AlertCodeSuspectedAIText.
	Code string `json:"code"`
	// Title is a human-readable title.
	Title string `json:"title"`
	// Message is a human-readable description.
	Message string `json:"message"`
	// HelpLink links to a description of the alert. Not every alert has one.
	HelpLink string `json:"helpLink"`
	// Severity is the importance of the alert, from 0 (very low) to 4 (very high).
	Severity int `json:"severity"`
	// AdditionalData is extra data as a JSON string; decode a suspected-ai-text alert's with AIDetectionResult.
	AdditionalData string `json:"additionalData"`
}

// AIDetectionAlert returns the first suspected-ai-text alert of the scan, or nil.
func (w *CompletedWebhook) AIDetectionAlert() *Alert {
	if w == nil || w.Notifications == nil {
		return nil
	}
	for i := range w.Notifications.Alerts {
		if w.Notifications.Alerts[i].Code == AlertCodeSuspectedAIText {
			return &w.Notifications.Alerts[i]
		}
	}
	return nil
}

// AIDetectionResult is AIDetectionAlert().AIDetectionResult(). It returns nil and no error when there is no
// alert or no result; use AIDetectionAlert to tell these apart.
func (w *CompletedWebhook) AIDetectionResult() (*AITextDetectionResponse, error) {
	return w.AIDetectionAlert().AIDetectionResult()
}

// trailingPadding is trimmed from AdditionalData before decoding.
const trailingPadding = "\x00\t\n\v\f\r "

// AIDetectionResult decodes the AdditionalData of a suspected-ai-text alert, ignoring trailing NULs and ASCII
// whitespace. It returns nil and no error for a nil or other alert, empty data, or JSON that is not an object
// (such as null), and an error for invalid JSON or a mistyped field; AdditionalData keeps the raw string.
func (a *Alert) AIDetectionResult() (*AITextDetectionResponse, error) {
	if a == nil || a.Code != AlertCodeSuspectedAIText {
		return nil, nil
	}
	data := []byte(strings.TrimRight(a.AdditionalData, trailingPadding))
	if len(data) == 0 {
		return nil, nil
	}
	// Only an object holds a result; check the first character rather than decode a large result twice.
	if t := bytes.TrimLeft(data, " \t\r\n"); len(t) == 0 || t[0] != '{' {
		if !json.Valid(data) {
			return nil, fmt.Errorf("copyleaks: decode suspected-ai-text additionalData: %w", invalidJSON(data))
		}
		return nil, nil
	}
	var res AITextDetectionResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("copyleaks: decode suspected-ai-text additionalData: %w", err)
	}
	return &res, nil
}

// ParseCompletedWebhook decodes the body of a completed webhook.
func ParseCompletedWebhook(body []byte) (*CompletedWebhook, error) {
	return parseWebhook[CompletedWebhook]("completed", body)
}

// ParseErrorWebhook decodes the body of an error webhook.
func ParseErrorWebhook(body []byte) (*ErrorWebhook, error) {
	return parseWebhook[ErrorWebhook]("error", body)
}

// ParseCreditsCheckedWebhook decodes the body of a creditsChecked webhook.
func ParseCreditsCheckedWebhook(body []byte) (*CreditsCheckedWebhook, error) {
	return parseWebhook[CreditsCheckedWebhook]("creditsChecked", body)
}

// ParseIndexedWebhook decodes the body of an indexed webhook.
func ParseIndexedWebhook(body []byte) (*IndexedWebhook, error) {
	return parseWebhook[IndexedWebhook]("indexed", body)
}

// ParseNewResultWebhook decodes the body of a new-result webhook.
func ParseNewResultWebhook(body []byte) (*NewResultWebhook, error) {
	return parseWebhook[NewResultWebhook]("new-result", body)
}

// ParseExportCompletedWebhook decodes the body of an export-completed webhook.
func ParseExportCompletedWebhook(body []byte) (*ExportCompletedWebhook, error) {
	return parseWebhook[ExportCompletedWebhook]("export-completed", body)
}

// invalidJSON returns the error that encoding/json reports for data, which is not valid JSON.
func invalidJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	return errors.New("invalid JSON")
}

// parseWebhook decodes a webhook body of the given kind.
func parseWebhook[T any](kind string, body []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, fmt.Errorf("copyleaks: parse %s webhook: %w", kind, err)
	}
	return &v, nil
}
