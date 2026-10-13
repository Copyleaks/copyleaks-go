package copyleaks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// AITextDetectionRequest is the body of DetectAIText.
type AITextDetectionRequest struct {
	// Text is the text to check, 255 to 100,000 characters (required).
	Text string `json:"text"`
	// Sandbox runs a free test check that returns mock results.
	Sandbox bool `json:"sandbox,omitempty"`
	// Language is the ISO 639-1 code of the text, such as "en". The server detects the
	// language when it is empty.
	Language string `json:"language,omitempty"`
	// Explain asks for an explanation of the detection (AI Logic), returned in
	// AITextDetectionResponse.Explain. Explanations are only returned for some languages.
	Explain bool `json:"explain,omitempty"`
	// Sensitivity controls the detection, from 1 to 3; higher values also catch AI text
	// that was edited more heavily. Nil keeps the server default, 2. Set it with Ptr.
	Sensitivity *int `json:"sensitivity,omitempty"`
}

// AITextDetectionResponse is the result of AI Text Detection. The same structure is
// decoded from the suspected-ai-text alert of a completed scan (see
// CompletedWebhook.AIDetectionResult).
//
// See https://docs.copyleaks.com/reference/data-types/ai-detector/ai-text-detector-response.
type AITextDetectionResponse struct {
	// ModelVersion is the version of the detection model.
	ModelVersion string `json:"modelVersion"`
	// Results are the classified sections of the text.
	Results []AITextDetectionResult `json:"results"`
	// Summary is the share of the text classified as human-written and as AI-generated.
	Summary *AITextDetectionSummary `json:"summary"`
	// TranslationProvider is the machine translation provider used before the detection.
	// 0 means that the text was not translated.
	TranslationProvider int `json:"translationProvider"`
	// Translation is the machine-translated English text, when the text was translated.
	Translation string `json:"translation"`
	// Explain explains the detection (AI Logic). It is filled when the request set
	// AITextDetectionRequest.Explain, for the languages that AI Logic supports.
	Explain *AITextDetectionExplain `json:"explain"`
	// ScannedDocument is general information about the check.
	ScannedDocument *ScannedDocument `json:"scannedDocument"`
	// ScanType is "sandbox" for sandbox checks and empty otherwise.
	ScanType string `json:"scanType"`
}

// UnmarshalJSON accepts modelVersion as a string or as a number.
func (r *AITextDetectionResponse) UnmarshalJSON(b []byte) error {
	type plain AITextDetectionResponse
	aux := struct {
		*plain
		ModelVersion json.RawMessage `json:"modelVersion"`
	}{plain: (*plain)(r)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	r.ModelVersion = ""
	if len(aux.ModelVersion) == 0 || string(aux.ModelVersion) == "null" {
		return nil
	}
	if err := json.Unmarshal(aux.ModelVersion, &r.ModelVersion); err == nil {
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(aux.ModelVersion, &n); err != nil {
		return fmt.Errorf("modelVersion: %w", err)
	}
	r.ModelVersion = n.String()
	return nil
}

// AITextDetectionResult is one classified section of the text.
type AITextDetectionResult struct {
	// Classification is 1 for human-written text and 2 for AI-generated text.
	Classification int `json:"classification"`
	// Probability is the confidence of the classification. Deprecated by the server; it
	// may be missing.
	Probability float64 `json:"probability"`
	// Matches are the locations of the section in the text.
	Matches []AITextDetectionMatch `json:"matches"`
}

// AITextDetectionMatch locates a classified section.
type AITextDetectionMatch struct {
	// Text locates the section in the plain text.
	Text *AITextLocation `json:"text"`
	// HTML locates the section in the HTML source, for HTML submissions.
	HTML *AITextLocation `json:"html"`
}

// AITextLocation holds character and word positions in the text.
type AITextLocation struct {
	// Chars are character positions.
	Chars *AITextPositions `json:"chars"`
	// Words are word positions.
	Words *AITextPositions `json:"words"`
}

// AITextPositions are segments of the text, as parallel start and length lists.
type AITextPositions struct {
	// Starts are the start positions of the segments.
	Starts []int `json:"starts"`
	// Lengths are the lengths of the segments, one per start position.
	Lengths []int `json:"lengths"`
	// GroupIDs are the HTML group IDs of the segments, for HTML positions.
	GroupIDs []int `json:"groupIds,omitempty"`
}

// AITextDetectionSummary is the share of the text classified as human-written and as
// AI-generated, from 0 to 1.
type AITextDetectionSummary struct {
	// Human is the share of human-written text.
	Human float64 `json:"human"`
	// AI is the share of AI-generated text.
	AI float64 `json:"ai"`
}

// AITextDetectionExplain explains the detection (AI Logic).
type AITextDetectionExplain struct {
	// Patterns are the AI patterns found in the text.
	Patterns *AITextDetectionPatterns `json:"patterns"`
}

// AITextDetectionPatterns are the AI patterns found in the text.
type AITextDetectionPatterns struct {
	// Statistics are statistics of each pattern.
	Statistics *AITextPatternStatistics `json:"statistics"`
	// Text locates the patterns in the plain text.
	Text *AITextLocation `json:"text"`
	// HTML locates the patterns in the HTML source, for HTML submissions.
	HTML *AITextLocation `json:"html"`
}

// AITextPatternStatistics are statistics of each AI pattern, as parallel lists.
type AITextPatternStatistics struct {
	// AICount is how often each pattern appears in AI-generated text.
	AICount []float64 `json:"aiCount"`
	// HumanCount is how often each pattern appears in human-written text.
	HumanCount []float64 `json:"humanCount"`
	// Proportion is the ratio between the AI and the human frequency of each pattern.
	Proportion []float64 `json:"proportion"`
	// Source is the source of each pattern: 1 for AI, 2 for a humanizer.
	Source []int `json:"source"`
}

// ScannedDocument is general information about an AI Text Detection, AI Image Detection,
// Grammar Checker or Text Moderation check.
type ScannedDocument struct {
	// ScanID is the scan ID given by you.
	ScanID string `json:"scanId"`
	// TotalWords is the number of words in the text.
	TotalWords int `json:"totalWords"`
	// TotalExcluded is the number of excluded words.
	TotalExcluded int `json:"totalExcluded"`
	// ActualCredits is the number of credits that the check used.
	ActualCredits int `json:"actualCredits"`
	// ExpectedCredits is the number of credits that the check was expected to use.
	ExpectedCredits int `json:"expectedCredits"`
	// CreationTime is when the check was created, in ISO 8601 format (UTC).
	CreationTime string `json:"creationTime"`
}

// DetectAIText checks whether a text was written by a human or generated by AI. The text
// must be 255 to 100,000 characters long. The scan ID must be 3 to 36 characters.
//
// See https://docs.copyleaks.com/reference/actions/writer-detector/check.
func (c *Client) DetectAIText(ctx context.Context, token *AuthToken, scanID string, req *AITextDetectionRequest) (*AITextDetectionResponse, error) {
	if err := validateID("scan ID", scanID); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, invalid("request is required")
	}
	if n := utf16Len(req.Text); n < minAITextLength || n > maxAITextLength {
		return nil, invalid(fmt.Sprintf("Text must be %d to %d characters, got %d", minAITextLength, maxAITextLength, n))
	}
	if req.Sensitivity != nil && (*req.Sensitivity < 1 || *req.Sensitivity > 3) {
		return nil, invalid(fmt.Sprintf("Sensitivity must be 1 to 3, got %d", *req.Sensitivity))
	}
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	var res AITextDetectionResponse
	err := c.do(ctx, apiRequest{
		method: http.MethodPost,
		path:   "/v2/writer-detector/" + segment(scanID) + "/check",
		token:  token,
		body:   req,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
