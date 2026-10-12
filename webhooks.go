package copyleaks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// AlertCodeSuspectedAIText is the code of the scan alert that carries the AI Text
// Detection result of a Plagiarism Checker scan that had ScanProperties.AIGeneratedText
// enabled.
const AlertCodeSuspectedAIText = "suspected-ai-text"

// CompletedWebhook is the payload of the completed webhook, which Copyleaks sends to
// the status webhook URL when a Plagiarism Checker scan finishes successfully.
//
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
}

// ErrorWebhook is the payload of the error webhook, sent when a scan ends with an error.
//
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

// CreditsCheckedWebhook is the payload of the creditsChecked webhook, sent for a scan
// submitted with SubmitActionCheckCredits.
//
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

// IndexedWebhook is the payload of the indexed webhook, sent for a scan submitted with
// SubmitActionIndex.
//
// See https://docs.copyleaks.com/reference/data-types/authenticity/webhooks/indexed.
type IndexedWebhook struct {
	// Status is the scan status: 3 for indexed.
	Status int `json:"status"`
	// DeveloperPayload is the ScanProperties.DeveloperPayload of the scan.
	DeveloperPayload string `json:"developerPayload"`
}

// NewResultWebhook is the payload of the new-result webhook, sent to
// SubmitWebhooks.NewResult each time a result is found while the scan still runs.
//
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
}

// ExportCompletedWebhook is the payload of the webhook that Copyleaks sends to
// ExportRequest.CompletionWebhook when an export is done. Check that Completed is true
// and that every task is healthy with a 2xx HTTPStatusCode.
//
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
	// SubmittedBy is the email of the user who added the document to the Private Cloud
	// Hub. Only Private Cloud Hub results have it.
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

// UnmarshalJSON decodes a score object, which the server sends. It also accepts a bare
// number, the shape that the API reference shows for the new-result webhook, as the
// AggregatedScore.
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
//
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
	// AdditionalData is extra data about the alert, as a JSON string. For a
	// suspected-ai-text alert, decode it with AIDetectionResult.
	AdditionalData string `json:"additionalData"`
}

// AIDetectionAlert returns the first suspected-ai-text alert of the scan, or nil when
// the completed webhook has none.
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

// AIDetectionResult decodes the AI Text Detection result of the scan from its
// suspected-ai-text alert. It is the same as calling AIDetectionResult on the alert that
// AIDetectionAlert returns.
//
// It returns nil and no error when there is no such alert, or when the alert carries no
// result; use AIDetectionAlert to tell these cases apart.
func (w *CompletedWebhook) AIDetectionResult() (*AITextDetectionResponse, error) {
	return w.AIDetectionAlert().AIDetectionResult()
}

// trailingPadding are the characters removed from the end of AdditionalData before it is
// decoded: NUL and ASCII whitespace.
const trailingPadding = "\x00\t\n\v\f\r "

// AIDetectionResult decodes the AdditionalData of a suspected-ai-text alert into the AI
// Text Detection result. Trailing NUL characters and ASCII whitespace are ignored.
//
// It returns nil and no error when a is nil, when the alert is not a suspected-ai-text
// alert, when AdditionalData is empty, or when it is valid JSON that is not an object,
// such as null. It returns an error when AdditionalData is not valid JSON or a field has
// an unexpected type; AdditionalData keeps the raw string.
func (a *Alert) AIDetectionResult() (*AITextDetectionResponse, error) {
	if a == nil || a.Code != AlertCodeSuspectedAIText {
		return nil, nil
	}
	data := []byte(strings.TrimRight(a.AdditionalData, trailingPadding))
	if len(data) == 0 {
		return nil, nil
	}
	var probe any
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("copyleaks: decode suspected-ai-text additionalData: %w", err)
	}
	if _, ok := probe.(map[string]any); !ok {
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

// parseWebhook decodes a webhook body of the given kind.
func parseWebhook[T any](kind string, body []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, fmt.Errorf("copyleaks: parse %s webhook: %w", kind, err)
	}
	return &v, nil
}
