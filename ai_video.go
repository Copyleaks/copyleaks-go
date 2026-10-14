package copyleaks

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
)

// AIVideoModel1Pro is the AI Video Detection model "ai-video-1-pro".
const AIVideoModel1Pro = "ai-video-1-pro"

// maxAsyncDeveloperPayload is the longest DeveloperPayload of an asynchronous AI Image
// or AI Video Detection submit.
const maxAsyncDeveloperPayload = 511

// aiVideoExtensions are the file extensions that AI Video Detection accepts.
var aiVideoExtensions = []string{".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv", ".wmv", ".mpg", ".m4v", ".3gp", ".mxf"}

// AIVideoSubmission is the body of SubmitAIVideo.
type AIVideoSubmission struct {
	// URL is the public address of the video to check (required).
	URL string `json:"url"`
	// Headers are the headers that Copyleaks sends to fetch URL, as [name, value] pairs.
	Headers [][]string `json:"headers,omitempty"`
	// Verb is the HTTP method that Copyleaks uses to fetch URL: "GET" (the server
	// default), "POST" or "PUT".
	Verb string `json:"verb,omitempty"`
	// Filename is the name of the video, up to 254 characters, ending with .mp4, .avi,
	// .mov, .mkv, .webm, .flv, .wmv, .mpg, .m4v, .3gp or .mxf (required).
	Filename string `json:"filename"`
	// Model is the detection model, such as AIVideoModel1Pro (required).
	Model string `json:"model"`
	// Webhooks is where Copyleaks sends the result (required); see ParseAIVideoWebhook.
	Webhooks ResultWebhook `json:"webhooks"`
	// Sandbox runs a free test check that returns mock results.
	Sandbox bool `json:"sandbox,omitempty"`
	// DeveloperPayload is a string of up to 511 characters that Copyleaks sends back with
	// the result.
	DeveloperPayload string `json:"developerPayload,omitempty"`
}

// SubmitAIVideo submits the video at a URL to AI Video Detection. The check runs in the
// background, and Copyleaks posts the result to req.Webhooks.URL; decode it with
// ParseAIVideoWebhook. The scan ID must be 3 to 36 characters.
//
// The video must be 2 seconds to 1 hour long, up to 512 MiB, at least 360x360 pixels and
// 16 frames per second. Copyleaks checks this after it downloads the video, and reports
// a failure to the webhook.
//
// See https://docs.copyleaks.com/reference/actions/ai-video-detector/submit.
func (c *Client) SubmitAIVideo(ctx context.Context, token *AuthToken, scanID string, req *AIVideoSubmission) error {
	if err := validateID("scan ID", scanID); err != nil {
		return err
	}
	if req == nil || req.URL == "" || req.Model == "" || req.Webhooks.URL == "" {
		return invalid("URL, Model and Webhooks.URL are required")
	}
	if n := utf16Len(req.Filename); n < 4 || n > 254 || !hasExtension(req.Filename, aiVideoExtensions) {
		return invalid("Filename must be up to 254 characters and end with .mp4, .avi, .mov, .mkv, .webm, .flv, .wmv, .mpg, .m4v, .3gp or .mxf")
	}
	if err := validateVerb(req.Verb); err != nil {
		return err
	}
	if utf16Len(req.DeveloperPayload) > maxAsyncDeveloperPayload {
		return invalid(fmt.Sprintf("DeveloperPayload must be at most %d characters", maxAsyncDeveloperPayload))
	}
	if err := VerifyAuthToken(token); err != nil {
		return err
	}
	return c.do(ctx, apiRequest{
		method:     http.MethodPost,
		path:       "/v1/ai-video-detector/" + segment(scanID) + "/submit",
		token:      token,
		body:       req,
		idempotent: true,
	}, nil)
}

// AIVideoDetectionResponse is the result of AI Video Detection, which Copyleaks posts to
// the webhook of SubmitAIVideo. Positions and lengths are in milliseconds.
//
// See https://docs.copyleaks.com/reference/data-types/ai-detector/ai-video-detection-response.
type AIVideoDetectionResponse struct {
	// Model is the detection model that was used.
	Model string `json:"model"`
	// AudioResult are the AI-generated segments of the audio track.
	AudioResult *AIVideoTrackResult `json:"audioResult"`
	// VisualResult are the AI-generated segments of the visual track.
	VisualResult *AIVideoTrackResult `json:"visualResult"`
	// Summary is the share of the video detected as AI-generated.
	Summary *AIVideoDetectionSummary `json:"summary"`
	// IsAIDetected reports whether the video contains AI-generated content. Use it rather
	// than interpreting the ratios of Summary.
	IsAIDetected bool `json:"isAiDetected"`
	// VideoInfo is information about the video.
	VideoInfo *AIVideoInfo `json:"videoInfo"`
	// ScannedVideo is general information about the check.
	ScannedVideo *ScannedVideo `json:"scannedVideo"`
	// DeveloperPayload is the DeveloperPayload of the submit, when the webhook carries it.
	DeveloperPayload string `json:"developerPayload,omitempty"`
	// Error tells why the check failed. Check it first: when it is set, the webhook
	// carries no result, only Error and DeveloperPayload.
	Error *AsyncError `json:"error,omitempty"`
}

// AIVideoTrackResult are the AI-generated segments of a track of a video, as parallel
// start and length lists, in milliseconds.
type AIVideoTrackResult struct {
	// Starts are the start positions of the AI-generated segments.
	Starts []int `json:"starts"`
	// Lengths are the durations of the AI-generated segments, one per start position.
	Lengths []int `json:"lengths"`
	// Exclude are the segments that were not scored, and do not count in the ratios.
	Exclude *AIVideoRanges `json:"exclude"`
}

// AIVideoRanges are segments of a video, as parallel start and length lists, in
// milliseconds.
type AIVideoRanges struct {
	// Starts are the start positions of the segments.
	Starts []int `json:"starts"`
	// Lengths are the durations of the segments, one per start position.
	Lengths []int `json:"lengths"`
}

// AIVideoDetectionSummary is the share of a video detected as AI-generated, from 0 to 1,
// over the scored segments only.
type AIVideoDetectionSummary struct {
	// AudioAIRatio is the share of the audible duration detected as AI-generated.
	AudioAIRatio float64 `json:"audioAIRatio"`
	// VisualAIRatio is the share of the visible duration detected as AI-generated.
	VisualAIRatio float64 `json:"visualAIRatio"`
	// OverallAIRatio is the share of the whole video detected as AI-generated, over both
	// tracks.
	OverallAIRatio float64 `json:"overallAIRatio"`
}

// AIVideoInfo is information about an analyzed video.
type AIVideoInfo struct {
	// Duration is the duration of the video, in seconds.
	Duration float64 `json:"duration"`
	// Metadata is metadata read from the video, such as C2PA provenance data, when it has
	// any.
	Metadata *AIVideoMetadata `json:"metadata"`
}

// AIVideoMetadata is metadata read from a video.
type AIVideoMetadata struct {
	// IssuedTime is when the video was created.
	IssuedTime string `json:"issuedTime"`
	// IssuedBy is the AI service or tool that created the video.
	IssuedBy string `json:"issuedBy"`
	// AppOrDeviceUsed is the application or device that created the video.
	AppOrDeviceUsed string `json:"appOrDeviceUsed"`
	// ContentSummary describes how the video was created.
	ContentSummary string `json:"contentSummary"`
}

// ScannedVideo is general information about an AI Video Detection check.
type ScannedVideo struct {
	// ScanID is the scan ID given by you.
	ScanID string `json:"scanId"`
	// ActualCredits is the number of credits that the check used.
	ActualCredits int `json:"actualCredits"`
	// ExpectedCredits is the number of credits that the check was expected to use.
	ExpectedCredits int `json:"expectedCredits"`
	// CreationTime is when the check was created, in ISO 8601 format (UTC).
	CreationTime string `json:"creationTime"`
}

// ParseAIVideoWebhook decodes the result of SubmitAIVideo that Copyleaks posts to the
// webhook. When the check failed, the result only has Error and DeveloperPayload, so
// check Error first.
func ParseAIVideoWebhook(body []byte) (*AIVideoDetectionResponse, error) {
	return parseWebhook[AIVideoDetectionResponse]("AI video", body)
}

// validateVerb checks the HTTP method that Copyleaks uses to fetch a URL: empty for the
// server default, or GET, POST or PUT in any case.
func validateVerb(verb string) error {
	switch strings.ToUpper(verb) {
	case "", http.MethodGet, http.MethodPost, http.MethodPut:
		return nil
	}
	return invalid(`Verb must be "GET", "POST" or "PUT"`)
}

// hasExtension reports whether filename ends with one of the extensions, ignoring case.
func hasExtension(filename string, extensions []string) bool {
	ext := strings.ToLower(path.Ext(filename))
	for _, e := range extensions {
		if ext == e {
			return true
		}
	}
	return false
}
