package copyleaks

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// This file models the files that Export uploads to your servers. Decode each upload
// with the matching Parse function.

// ExportedResult is a result file that Export uploads for each of ExportRequest.Results:
// the text of a matching source and where it matches the scanned document.
//
// See https://docs.copyleaks.com/reference/data-types/authenticity/results/new-plagiarism-result.
type ExportedResult struct {
	// Statistics count the matching words, by kind of match.
	Statistics *ExportedResultStatistics `json:"statistics"`
	// Text is the source as plain text, with the matches.
	Text *ExportedResultText `json:"text"`
	// HTML is the source as HTML, with the matches, when the scan was submitted with
	// ScanProperties.IncludeHTML. Otherwise it is empty.
	HTML *ExportedResultHTML `json:"html"`
}

// ExportedResultStatistics count the matching words of a result, by kind of match.
type ExportedResultStatistics struct {
	// Identical is the number of identical words.
	Identical int `json:"identical"`
	// MinorChanges is the number of nearly identical words, such as slow and slowly.
	MinorChanges int `json:"minorChanges"`
	// RelatedMeaning is the number of paraphrased words.
	RelatedMeaning int `json:"relatedMeaning"`
}

// ExportedResultText is the plain text of a result source, with the matches.
type ExportedResultText struct {
	// Value is the text of the source.
	Value string `json:"value"`
	// Pages are the pages of the text.
	Pages *TextPages `json:"pages"`
	// Comparison locates the matches, by kind of match.
	Comparison *ResultComparison `json:"comparison"`
}

// ExportedResultHTML is the HTML of a result source, with the matches.
type ExportedResultHTML struct {
	// Value is the HTML of the source.
	Value string `json:"value"`
	// Comparison locates the matches, by kind of match.
	Comparison *ResultComparison `json:"comparison"`
}

// TextPages are the pages of a text.
type TextPages struct {
	// StartPosition are the character positions where the pages start. The first one is
	// the start of the first page.
	StartPosition []int `json:"startPosition"`
}

// ResultComparison locates the matches between a result source and the scanned
// document, by kind of match.
type ResultComparison struct {
	// Identical are the identical matches.
	Identical *ComparisonMatches `json:"identical"`
	// MinorChanges are the nearly identical matches.
	MinorChanges *ComparisonMatches `json:"minorChanges"`
	// RelatedMeaning are the paraphrased matches.
	RelatedMeaning *ComparisonMatches `json:"relatedMeaning"`
}

// ComparisonMatches are matches of one kind, located in the source and in the scanned
// document. The match at index i is at index i of every list.
type ComparisonMatches struct {
	// GroupID are the HTML group IDs of the matches, in the HTML comparison only.
	GroupID []int `json:"groupId,omitempty"`
	// Source locates the matches in the result source.
	Source *ComparisonLocation `json:"source"`
	// Suspected locates the matches in the scanned document.
	Suspected *ComparisonLocation `json:"suspected"`
}

// ComparisonLocation locates matches by character and by word.
type ComparisonLocation struct {
	// Chars are the character positions of the matches.
	Chars *TextRanges `json:"chars"`
	// Words are the word positions of the matches.
	Words *TextRanges `json:"words"`
}

// TextRanges are ranges of a text, as parallel start and length lists.
type TextRanges struct {
	// Starts are the start positions of the ranges.
	Starts []int `json:"starts"`
	// Lengths are the lengths of the ranges, one per start position.
	Lengths []int `json:"lengths"`
}

// ExportedCrawledVersion is the crawled version that Export uploads for
// ExportRequest.CrawledVersion: the submitted document as Copyleaks read it.
//
// See https://docs.copyleaks.com/reference/data-types/authenticity/results/crawled-version.
type ExportedCrawledVersion struct {
	// Metadata counts the words of the document.
	Metadata *CrawledVersionMetadata `json:"metadata"`
	// Text is the document as plain text.
	Text *CrawledVersionText `json:"text"`
	// HTML is the document as HTML, when the scan was submitted with
	// ScanProperties.IncludeHTML.
	HTML *CrawledVersionHTML `json:"html"`
	// ReferencesValidation is the result of the Citation Checker, for a scan submitted
	// with ScanProperties.References.Validate.
	ReferencesValidation *ReferencesValidation `json:"referencesValidation"`
}

// CrawledVersionMetadata counts the words of a crawled document.
type CrawledVersionMetadata struct {
	// Words is the number of words.
	Words int `json:"words"`
	// Excluded is the number of excluded words.
	Excluded int `json:"excluded"`
}

// CrawledVersionText is a crawled document as plain text.
type CrawledVersionText struct {
	// Value is the text.
	Value string `json:"value"`
	// Exclude are the parts of the text that were not scanned.
	Exclude *ExcludedRanges `json:"exclude"`
	// Pages are the pages of the text.
	Pages *TextPages `json:"pages"`
}

// CrawledVersionHTML is a crawled document as HTML.
type CrawledVersionHTML struct {
	// Value is the HTML.
	Value string `json:"value"`
	// Exclude are the parts of the HTML that were not scanned.
	Exclude *ExcludedRanges `json:"exclude"`
}

// ExcludedRanges are the parts of a document that were not scanned, as parallel lists:
// the part at index i is at index i of every list.
type ExcludedRanges struct {
	// Starts are the start positions of the parts.
	Starts []int `json:"starts"`
	// Lengths are the lengths of the parts, in characters.
	Lengths []int `json:"lengths"`
	// Reasons are the reasons why the parts were excluded.
	Reasons []int `json:"reasons"`
	// GroupIDs are the HTML group IDs of the parts, in the HTML version only.
	GroupIDs []int `json:"groupIds,omitempty"`
}

// ReferenceValidation is the validation of one reference by the Citation Checker.
type ReferenceValidation struct {
	// Type is "academic" for a reference checked against the Copyleaks academic citation
	// index, and "nonAcademic" for one checked against the page that it cites.
	Type string `json:"type"`
	// TextRanges locate the reference in the text.
	TextRanges *TextRanges `json:"textRanges"`
	// HTMLRanges locate the reference in the HTML. They are empty for plain text scans.
	HTMLRanges *HTMLRanges `json:"htmlRanges"`
	// Parsed are the fields that Copyleaks read from the reference.
	Parsed *ParsedReference `json:"parsed"`
	// Suggestions are the sources found for the reference. It is empty when none was
	// found.
	Suggestions []ReferenceSuggestion `json:"suggestions"`
}

// HTMLRanges are ranges of an HTML document, as parallel lists.
type HTMLRanges struct {
	// Starts are the start positions of the ranges.
	Starts []int `json:"starts"`
	// Lengths are the lengths of the ranges, in characters.
	Lengths []int `json:"lengths"`
	// GroupIDs group the ranges of one reference.
	GroupIDs []int `json:"groupIds"`
}

// UnmarshalJSON decodes the ranges. It also accepts an empty list, which stands for no
// ranges.
func (r *HTMLRanges) UnmarshalJSON(b []byte) error {
	if t := bytes.TrimSpace(b); len(t) > 0 && t[0] == '[' {
		var list []json.RawMessage
		if err := json.Unmarshal(t, &list); err != nil {
			return err
		}
		*r = HTMLRanges{}
		return nil
	}
	type plain HTMLRanges
	return json.Unmarshal(b, (*plain)(r))
}

// ParsedReference are the fields that the Citation Checker read from a reference.
type ParsedReference struct {
	// Title is the cited title.
	Title string `json:"title"`
	// Authors are the cited authors.
	Authors []string `json:"authors"`
	// Year is the cited year.
	Year int `json:"year"`
	// Journal is the cited journal or publication. It may be empty.
	Journal string `json:"journal"`
	// URL is the cited link, for non-academic references.
	URL string `json:"url"`
}

// ReferenceSuggestion is a source found for a reference.
type ReferenceSuggestion struct {
	// URL is the address of the source.
	URL string `json:"url"`
	// DOI is the DOI of the source. It may be empty.
	DOI string `json:"doi"`
	// Title is the title of the source.
	Title string `json:"title"`
	// Authors are the authors of the source.
	Authors []string `json:"authors"`
	// PublishedYear is the year the source was published, or 0 when unknown.
	PublishedYear int `json:"publishedYear"`
	// ModifiedYear is the year the source was last modified, or 0 when unknown.
	ModifiedYear int `json:"modifiedYear"`
	// Journal is the journal or publication of the source. It may be empty.
	Journal string `json:"journal"`
	// Signals report which fields of the reference the source corroborates.
	Signals *ReferenceSignals `json:"signals"`
}

// ReferenceSignals report which fields of a reference a source corroborates. A nil field
// was not checked and does not count against the reference.
type ReferenceSignals struct {
	// Title reports whether the title matches.
	Title *bool `json:"title"`
	// Year reports whether the cited year is the published or the modified year of the
	// source.
	Year *bool `json:"year"`
	// Authors reports whether the authors match.
	Authors *bool `json:"authors"`
}

// ExportedOverview is the Gen-AI Overview that Export uploads for ExportRequest.Overview.
//
// See https://docs.copyleaks.com/reference/data-types/authenticity/results/ai-overview.
type ExportedOverview struct {
	// Overview is the overview of the scan, in Markdown.
	Overview string `json:"overview"`
	// ModelVersion is the version of the model that wrote the overview.
	ModelVersion string `json:"modelVersion"`
}

// ParseExportedResult decodes a result file that Export uploaded for one of
// ExportRequest.Results.
func ParseExportedResult(body []byte) (*ExportedResult, error) {
	return parseExport[ExportedResult]("result", body)
}

// ParseExportedCrawledVersion decodes the crawled version that Export uploaded for
// ExportRequest.CrawledVersion.
func ParseExportedCrawledVersion(body []byte) (*ExportedCrawledVersion, error) {
	return parseExport[ExportedCrawledVersion]("crawled version", body)
}

// ParseExportedAIDetection decodes the AI Text Detection report that Export uploaded for
// ExportRequest.AIDetection. It has the shape of the DetectAIText response.
//
// See https://docs.copyleaks.com/reference/data-types/authenticity/results/ai-detection.
func ParseExportedAIDetection(body []byte) (*AITextDetectionResponse, error) {
	return parseExport[AITextDetectionResponse]("AI detection", body)
}

// ParseExportedWritingFeedback decodes the Grammar Checker report that Export uploaded
// for ExportRequest.WritingFeedback. It has the shape of the CheckGrammar response.
//
// See https://docs.copyleaks.com/reference/data-types/writing/writing-assistant.
func ParseExportedWritingFeedback(body []byte) (*GrammarCheckResponse, error) {
	return parseExport[GrammarCheckResponse]("writing feedback", body)
}

// ParseExportedOverview decodes the Gen-AI Overview that Export uploaded for
// ExportRequest.Overview.
func ParseExportedOverview(body []byte) (*ExportedOverview, error) {
	return parseExport[ExportedOverview]("overview", body)
}

// parseExport decodes an exported file of the given kind.
func parseExport[T any](kind string, body []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, fmt.Errorf("copyleaks: parse exported %s: %w", kind, err)
	}
	return &v, nil
}
