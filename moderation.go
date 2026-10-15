package copyleaks

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Copyleaks/copyleaks-go/internal/validate"
)

// Text Moderation label IDs; see https://docs.copyleaks.com/reference/data-types/moderation/text-moderation-labels.
const (
	// TextModerationLabelAdultV1 flags adult content.
	TextModerationLabelAdultV1 = "adult-v1"
	// TextModerationLabelToxicV1 flags toxic content.
	TextModerationLabelToxicV1 = "toxic-v1"
	// TextModerationLabelViolentV1 flags violent content.
	TextModerationLabelViolentV1 = "violent-v1"
	// TextModerationLabelProfanityV1 flags profanity.
	TextModerationLabelProfanityV1 = "profanity-v1"
	// TextModerationLabelSelfHarmV1 flags self-harm content.
	TextModerationLabelSelfHarmV1 = "self-harm-v1"
	// TextModerationLabelHarassmentV1 flags harassment.
	TextModerationLabelHarassmentV1 = "harassment-v1"
	// TextModerationLabelHateSpeechV1 flags hate speech.
	TextModerationLabelHateSpeechV1 = "hate-speech-v1"
	// TextModerationLabelDrugsV1 flags drug-related content.
	TextModerationLabelDrugsV1 = "drugs-v1"
	// TextModerationLabelFirearmsV1 flags firearm-related content.
	TextModerationLabelFirearmsV1 = "firearms-v1"
	// TextModerationLabelCybersecurityV1 flags cybersecurity threats.
	TextModerationLabelCybersecurityV1 = "cybersecurity-v1"
)

// TextModerationLanguageEnglish is the code of English, the only language that Text Moderation supports.
const TextModerationLanguageEnglish = "en"

// maxTextModerationLabels is the most labels that one Text Moderation request accepts.
const maxTextModerationLabels = 16

// Text Moderation text length limits, in UTF-16 code units.
const (
	minTextModerationLength = 2
	maxTextModerationLength = 25000
)

// TextModerationRequest is the body of ModerateText.
type TextModerationRequest struct {
	// Text is the text to moderate, 2 to 25,000 characters (required).
	Text string `json:"text"`
	// Sandbox runs a free test check that returns mock results.
	Sandbox bool `json:"sandbox,omitempty"`
	// Language is the language code: TextModerationLanguageEnglish, or empty to let the server detect it.
	Language string `json:"language,omitempty"`
	// Labels are 1 to 16 categories to check for (required); a flagged segment's label value indexes this list.
	Labels []TextModerationLabel `json:"labels"`
}

// TextModerationLabel is a category to check the text for, such as TextModerationLabelToxicV1.
type TextModerationLabel struct {
	// ID is the label ID.
	ID string `json:"id"`
}

// TextModerationResponse is the result of Text Moderation.
type TextModerationResponse struct {
	// ModelVersion is the version of the moderation model.
	ModelVersion string `json:"modelVersion"`
	// Moderations are the flagged segments of the text.
	Moderations *TextModerations `json:"moderations"`
	// Legend maps the label values of Moderations to label IDs.
	Legend []TextModerationLegend `json:"legend"`
	// ScannedDocument is general information about the check.
	ScannedDocument *ScannedDocument `json:"scannedDocument"`
}

// TextModerations are the flagged segments of a text.
type TextModerations struct {
	// Text holds the flagged segments of the plain text.
	Text *TextModerationText `json:"text"`
}

// TextModerationText holds the flagged segments of the plain text.
type TextModerationText struct {
	// Chars are the flagged segments, by character position.
	Chars *TextModerationChars `json:"chars"`
}

// TextModerationChars are flagged segments as parallel lists: the segment at index i
// starts at Starts[i], is Lengths[i] characters long and has label value Labels[i].
type TextModerationChars struct {
	// Labels are the label values of the segments; see TextModerationResponse.Legend.
	Labels []int `json:"labels"`
	// Starts are the start positions of the segments.
	Starts []int `json:"starts"`
	// Lengths are the lengths of the segments.
	Lengths []int `json:"lengths"`
}

// TextModerationLegend maps a label value to a label ID.
type TextModerationLegend struct {
	// Index is the label value used in TextModerationChars.Labels.
	Index int `json:"index"`
	// ID is the label ID, such as TextModerationLabelToxicV1.
	ID string `json:"id"`
}

// ModerateText flags hateful, explicit, toxic or otherwise risky content in a text. Scan ID: 3 to 36 characters.
// See https://docs.copyleaks.com/reference/actions/text-moderation/check.
func (c *Client) ModerateText(ctx context.Context, token *AuthToken, scanID string, req *TextModerationRequest) (*TextModerationResponse, error) {
	if err := validate.ID("scan ID", scanID); err != nil {
		return nil, invalid(err.Error())
	}
	if req == nil {
		return nil, invalid("request is required")
	}
	if n := validate.UTF16Len(req.Text); n < minTextModerationLength || n > maxTextModerationLength {
		return nil, invalid(fmt.Sprintf("Text must be %d to %d characters, got %d", minTextModerationLength, maxTextModerationLength, n))
	}
	if n := len(req.Labels); n == 0 || n > maxTextModerationLabels {
		return nil, invalid(fmt.Sprintf("Labels must list 1 to %d labels, got %d", maxTextModerationLabels, n))
	}
	for _, l := range req.Labels {
		if l.ID == "" {
			return nil, invalid("each of Labels needs an ID")
		}
	}
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	var res TextModerationResponse
	err := c.do(ctx, apiRequest{
		method: http.MethodPost,
		path:   "/v1/text-moderation/" + segment(scanID) + "/check",
		token:  token,
		body:   req,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
