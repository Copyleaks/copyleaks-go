package copyleaks

import (
	"context"
	"net/http"

	"github.com/Copyleaks/copyleaks-go/internal/validate"
)

// GrammarCheckRequest is the body of CheckGrammar.
type GrammarCheckRequest struct {
	// Text is the text to check, up to 25,000 characters (required).
	Text string `json:"text"`
	// Sandbox runs a free test check that returns mock results.
	Sandbox bool `json:"sandbox,omitempty"`
	// Language is the language code of the text, such as "en"; empty lets the server detect it.
	Language string `json:"language,omitempty"`
	// Score sets the weight of each correction category in the overall score.
	Score *GrammarScoreWeights `json:"score,omitempty"`
}

// GrammarScoreWeights weighs each correction category in the overall score, 0.0 to 1.0; nil keeps 1.0.
type GrammarScoreWeights struct {
	// GrammarScoreWeight is the weight of grammar corrections.
	GrammarScoreWeight *float64 `json:"grammarScoreWeight,omitempty"`
	// MechanicsScoreWeight is the weight of mechanics corrections.
	MechanicsScoreWeight *float64 `json:"mechanicsScoreWeight,omitempty"`
	// SentenceStructureScoreWeight is the weight of sentence structure corrections.
	SentenceStructureScoreWeight *float64 `json:"sentenceStructureScoreWeight,omitempty"`
	// WordChoiceScoreWeight is the weight of word choice corrections.
	WordChoiceScoreWeight *float64 `json:"wordChoiceScoreWeight,omitempty"`
}

// GrammarCheckResponse is the result of the Grammar Checker.
// See https://docs.copyleaks.com/reference/data-types/writing/writing-assistant.
type GrammarCheckResponse struct {
	// Score holds the writing scores, the readability and statistics of the text.
	Score *GrammarScore `json:"score"`
	// Corrections are the suggested corrections.
	Corrections *GrammarCorrections `json:"corrections"`
	// ScannedDocument is general information about the check.
	ScannedDocument *ScannedDocument `json:"scannedDocument"`
	// ScanType is "sandbox" for sandbox checks and empty otherwise.
	ScanType string `json:"scanType"`
}

// GrammarScore holds the writing scores, the readability and statistics of a text.
type GrammarScore struct {
	// Corrections are the correction counts and scores of each category, and the overall score.
	Corrections *GrammarScoreCorrections `json:"corrections"`
	// Readability is the readability of the text.
	Readability *GrammarReadability `json:"readability"`
	// Statistics are statistics of the text.
	Statistics *GrammarStatistics `json:"statistics"`
}

// GrammarScoreCorrections are the correction counts and scores of each category.
type GrammarScoreCorrections struct {
	// GrammarCorrectionsCount is the number of grammar corrections.
	GrammarCorrectionsCount int `json:"grammarCorrectionsCount"`
	// GrammarCorrectionsScore is the grammar score.
	GrammarCorrectionsScore int `json:"grammarCorrectionsScore"`
	// GrammarScoreWeight is the weight of the grammar score in the overall score.
	GrammarScoreWeight float64 `json:"grammarScoreWeight"`
	// MechanicsCorrectionsCount is the number of mechanics corrections.
	MechanicsCorrectionsCount int `json:"mechanicsCorrectionsCount"`
	// MechanicsCorrectionsScore is the mechanics score.
	MechanicsCorrectionsScore int `json:"mechanicsCorrectionsScore"`
	// MechanicsScoreWeight is the weight of the mechanics score in the overall score.
	MechanicsScoreWeight float64 `json:"mechanicsScoreWeight"`
	// SentenceStructureCorrectionsCount is the number of sentence structure corrections.
	SentenceStructureCorrectionsCount int `json:"sentenceStructureCorrectionsCount"`
	// SentenceStructureCorrectionsScore is the sentence structure score.
	SentenceStructureCorrectionsScore int `json:"sentenceStructureCorrectionsScore"`
	// SentenceStructureScoreWeight is the weight of the sentence structure score in the overall score.
	SentenceStructureScoreWeight float64 `json:"sentenceStructureScoreWeight"`
	// WordChoiceCorrectionsCount is the number of word choice corrections.
	WordChoiceCorrectionsCount int `json:"wordChoiceCorrectionsCount"`
	// WordChoiceCorrectionsScore is the word choice score.
	WordChoiceCorrectionsScore int `json:"wordChoiceCorrectionsScore"`
	// WordChoiceScoreWeight is the weight of the word choice score in the overall score.
	WordChoiceScoreWeight float64 `json:"wordChoiceScoreWeight"`
	// OverallScore is the overall writing score.
	OverallScore int `json:"overallScore"`
}

// GrammarReadability is the readability of a text.
type GrammarReadability struct {
	// Score is the readability score.
	Score int `json:"score"`
	// ReadabilityLevel is the readability level.
	ReadabilityLevel int `json:"readabilityLevel"`
	// ReadabilityLevelText is the readability level as text, such as "10th to 12th Grader".
	ReadabilityLevelText string `json:"readabilityLevelText"`
	// ReadabilityLevelDescription describes the readability level, such as "Fairly difficult to read".
	ReadabilityLevelDescription string `json:"readabilityLevelDescription"`
}

// GrammarStatistics are statistics of a text.
type GrammarStatistics struct {
	// SentenceCount is the number of sentences.
	SentenceCount int `json:"sentenceCount"`
	// AverageSentenceLength is the average sentence length, in words.
	AverageSentenceLength float64 `json:"averageSentenceLength"`
	// AverageWordLength is the average word length, in characters.
	AverageWordLength float64 `json:"averageWordLength"`
	// ReadingTimeSeconds is the estimated reading time, in seconds.
	ReadingTimeSeconds float64 `json:"readingTimeSeconds"`
	// SpeakingTimeSeconds is the estimated speaking time, in seconds.
	SpeakingTimeSeconds float64 `json:"speakingTimeSeconds"`
}

// GrammarCorrections are the corrections suggested for a text.
type GrammarCorrections struct {
	// Text holds the corrections of the plain text.
	Text *GrammarCorrectionsText `json:"text"`
}

// GrammarCorrectionsText holds the corrections of the plain text.
type GrammarCorrectionsText struct {
	// Chars are the corrections, by character position.
	Chars *GrammarCorrectionRanges `json:"chars"`
}

// GrammarCorrectionRanges are corrections as parallel lists: the correction at index i
// replaces Lengths[i] characters at Starts[i] with OperationTexts[i].
type GrammarCorrectionRanges struct {
	// Types are the correction type IDs; see Client.GrammarCorrectionTypes.
	Types []int `json:"types"`
	// Starts are the start positions of the corrected segments.
	Starts []int `json:"starts"`
	// Lengths are the lengths of the corrected segments.
	Lengths []int `json:"lengths"`
	// OperationTexts are the suggested replacement texts.
	OperationTexts []string `json:"operationTexts"`
}

// CheckGrammar returns grammar, spelling and sentence structure corrections. Scan ID: 3 to 36 characters.
// See https://docs.copyleaks.com/reference/actions/writing-assistant/check.
func (c *Client) CheckGrammar(ctx context.Context, token *AuthToken, scanID string, req *GrammarCheckRequest) (*GrammarCheckResponse, error) {
	if err := validate.ID("scan ID", scanID); err != nil {
		return nil, invalid(err.Error())
	}
	if req == nil || req.Text == "" {
		return nil, invalid("Text is required")
	}
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	var res GrammarCheckResponse
	err := c.do(ctx, apiRequest{
		method: http.MethodPost,
		path:   "/v1/writing-feedback/" + segment(scanID) + "/check",
		token:  token,
		body:   req,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// CorrectionTypes are the correction types of the Grammar Checker.
type CorrectionTypes struct {
	// CorrectionTypes are the correction types.
	CorrectionTypes []CorrectionType `json:"correctionTypes"`
}

// CorrectionType is one type of Grammar Checker correction.
type CorrectionType struct {
	// ID is the correction type ID, as used in GrammarCorrectionRanges.Types.
	ID int `json:"id"`
	// Category is the ID of the correction category.
	Category int `json:"category"`
	// Title is the name of the correction type.
	Title string `json:"title"`
	// Message is a short message about the correction.
	Message string `json:"message"`
	// Description describes the correction type.
	Description string `json:"description"`
}

// GrammarCorrectionTypes returns the Grammar Checker correction types, with texts in language, such as "en".
// No login needed. See https://docs.copyleaks.com/reference/actions/writing-assistant/correction-types.
func (c *Client) GrammarCorrectionTypes(ctx context.Context, language string) (*CorrectionTypes, error) {
	if language == "" {
		return nil, invalid("language is required")
	}
	var res CorrectionTypes
	err := c.do(ctx, apiRequest{
		method: http.MethodGet,
		path:   "/v1/writing-feedback/correction-types/" + segment(language),
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
