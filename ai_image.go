package copyleaks

import (
	"context"
	"fmt"
	"net/http"
)

// AIImageModel1Ultra is the AI Image Detection model "ai-image-1-ultra". It produces a
// mask of the regions detected as AI-generated.
const AIImageModel1Ultra = "ai-image-1-ultra"

// The values of AIImageDetectionRequest.MaskType, the kind of mask that AI Image
// Detection returns.
const (
	// AIImageMaskBinary returns a run-length encoded mask of the AI-generated regions in
	// AIImageDetectionResponse.Result. It is the default.
	AIImageMaskBinary = "binary"
	// AIImageMaskHeatmap also returns a confidence band, from 0 to 4, for each run of the
	// mask, in AIImageDetectionResult.Labels.
	AIImageMaskHeatmap = "heatmap"
	// AIImageMaskNone returns no mask: AIImageDetectionResponse.Result is nil.
	AIImageMaskNone = "none"
)

// maxAIImageObjects and maxAIImageObjectLength are the limits of the object detection
// list of AI Image Detection.
const (
	maxAIImageObjects      = 3
	maxAIImageObjectLength = 32
)

// AIImageObject is an object to locate in an image, such as "face". See
// AIImageDetectionRequest.ObjectDetection.
type AIImageObject struct {
	// Object names the object, 1 to 32 characters.
	Object string `json:"object"`
}

// AIImageDetectionRequest is the body of DetectAIImage.
type AIImageDetectionRequest struct {
	// Base64 is the image, base64-encoded (required). The image must be at least 512x512
	// pixels and at most 27 megapixels, under 32 MB, in PNG, JPEG, BMP, WebP, HEIC/HEIF or
	// AVIF format. The server checks the size.
	Base64 string `json:"base64"`
	// Filename is the name of the image, 5 to 254 characters, ending with .png, .jpg,
	// .jpeg, .bmp, .webp, .heic, .heif or .avif (required).
	Filename string `json:"filename"`
	// Model is the detection model, such as AIImageModel1Ultra (required).
	Model string `json:"model"`
	// Sandbox runs a free test check that returns mock results.
	Sandbox bool `json:"sandbox,omitempty"`
	// MaskType is the kind of mask to return: AIImageMaskBinary (the server default),
	// AIImageMaskHeatmap or AIImageMaskNone.
	MaskType string `json:"maskType,omitempty"`
	// ObjectDetection lists up to 3 objects, such as "face" or "hand", to locate in the
	// image: &[]copyleaks.AIImageObject{{Object: "face"}}. Each one found is returned,
	// with its mask, in AIImageDetectionResponse.DetectedObjects. It does not change the
	// cost. It is a pointer, so that AIImageDetectionRequest stays comparable.
	ObjectDetection *[]AIImageObject `json:"objectDetection,omitempty"`
}

// AIImageDetectionResponse is the result of AI Image Detection.
//
// See https://docs.copyleaks.com/reference/data-types/ai-detector/ai-image-detection-response.
type AIImageDetectionResponse struct {
	// Model is the detection model that was used.
	Model string `json:"model"`
	// Result is the run-length encoded mask of the AI-generated regions, over the image
	// pixels flattened row by row.
	Result *AIImageDetectionResult `json:"result"`
	// Summary is the share of pixels detected as human-made and as AI-generated.
	Summary *AIImageDetectionSummary `json:"summary"`
	// IsAIDetected reports whether AI-generated content was detected in the image.
	IsAIDetected bool `json:"isAiDetected"`
	// ImageInfo is information about the image.
	ImageInfo *AIImageInfo `json:"imageInfo"`
	// ScannedDocument is general information about the check. TotalWords and
	// TotalExcluded are not used.
	ScannedDocument *ScannedDocument `json:"scannedDocument"`
	// DetectedObjects are the objects of the request's ObjectDetection list that were
	// found in the image, each with its mask. Objects that were not found are left out,
	// so look them up by Object. It is nil when the request had no ObjectDetection list.
	// It is a pointer, so that AIImageDetectionResponse stays comparable.
	DetectedObjects *[]AIImageDetectedObject `json:"detectedObjects"`
	// DeveloperPayload is the DeveloperPayload of an AIImageAsyncSubmission, when the
	// webhook carries it.
	DeveloperPayload string `json:"developerPayload,omitempty"`
	// Error tells why a check submitted with SubmitAIImageAsync failed. Check it first:
	// when it is set, the webhook carries no result, only Error and DeveloperPayload.
	// DetectAIImage and DetectAIImageUpload return an *APIError instead, and never set
	// it.
	Error *WebhookError `json:"error,omitempty"`
}

// AIImageDetectionResult is a run-length encoded mask, as parallel start and length lists.
type AIImageDetectionResult struct {
	// Starts are the start positions of the AI-generated runs.
	Starts []int `json:"starts"`
	// Lengths are the lengths of the AI-generated runs, one per start position.
	Lengths []int `json:"lengths"`
	// Labels are the confidence bands of the runs, from 0 (lowest) to 4 (highest), one
	// per start position. They are only returned for AIImageMaskHeatmap.
	Labels []int `json:"labels,omitempty"`
}

// AIImageDetectedObject is an object found in an image.
type AIImageDetectedObject struct {
	// Object is the name of the object, as in the request.
	Object string `json:"object"`
	// Mask is the run-length encoded mask of the object, over the image pixels flattened
	// row by row.
	Mask *AIImageDetectionResult `json:"mask"`
}

// AIImageDetectionSummary is the share of pixels detected as human-made and as
// AI-generated, from 0 to 1.
type AIImageDetectionSummary struct {
	// Human is the share of human-made pixels.
	Human float64 `json:"human"`
	// AI is the share of AI-generated pixels.
	AI float64 `json:"ai"`
}

// AIImageInfo is information about an analyzed image.
type AIImageInfo struct {
	// Shape is the size of the image.
	Shape *AIImageShape `json:"shape"`
	// Metadata is metadata read from the image, when it has any.
	Metadata *AIImageMetadata `json:"metadata"`
}

// AIImageShape is the size of an image, in pixels.
type AIImageShape struct {
	// Height is the height of the image.
	Height int `json:"height"`
	// Width is the width of the image.
	Width int `json:"width"`
}

// AIImageMetadata is metadata read from an image.
type AIImageMetadata struct {
	// IssuedTime is when the image was created.
	IssuedTime string `json:"issuedTime"`
	// IssuedBy is the AI service or tool that created the image.
	IssuedBy string `json:"issuedBy"`
	// AppOrDeviceUsed is the application or device that created the image.
	AppOrDeviceUsed string `json:"appOrDeviceUsed"`
	// ContentSummary describes how the content was created.
	ContentSummary string `json:"contentSummary"`
}

// DetectAIImage checks whether an image, or part of it, was generated by AI. The scan ID
// must be 3 to 36 characters.
//
// See https://docs.copyleaks.com/reference/actions/ai-image-detector/check.
func (c *Client) DetectAIImage(ctx context.Context, token *AuthToken, scanID string, req *AIImageDetectionRequest) (*AIImageDetectionResponse, error) {
	if err := validateID("scan ID", scanID); err != nil {
		return nil, err
	}
	if req == nil || req.Base64 == "" || req.Model == "" {
		return nil, invalid("Base64 and Model are required")
	}
	if err := validateAIImageOptions(req.Filename, req.MaskType, valueOf(req.ObjectDetection)); err != nil {
		return nil, err
	}
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	head := *req
	head.Base64 = ""
	body, err := base64Body(&head, req.Base64, nil)
	if err != nil {
		return nil, fmt.Errorf("copyleaks: encode POST /v1/ai-image-detector request: %w", err)
	}
	var res AIImageDetectionResponse
	err = c.do(ctx, apiRequest{
		method:      http.MethodPost,
		path:        "/v1/ai-image-detector/" + segment(scanID) + "/check",
		token:       token,
		rawBody:     body,
		contentType: "application/json",
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// AIImageUploadRequest is the request of DetectAIImageUpload. It holds the same options
// as AIImageDetectionRequest, with the image as it is instead of base64-encoded.
type AIImageUploadRequest struct {
	// Image is the image file (required). The image must be at least 512x512 pixels and
	// at most 27 megapixels, under 32 MB, in PNG, JPEG, BMP, WebP, HEIC/HEIF or AVIF
	// format. The server checks the size.
	Image []byte
	// Filename is the name of the image, 5 to 254 characters, ending with .png, .jpg,
	// .jpeg, .bmp, .webp, .heic, .heif or .avif (required).
	Filename string
	// Model is the detection model, such as AIImageModel1Ultra (required).
	Model string
	// Sandbox runs a free test check that returns mock results.
	Sandbox bool
	// MaskType is the kind of mask to return: AIImageMaskBinary (the server default),
	// AIImageMaskHeatmap or AIImageMaskNone.
	MaskType string
	// ObjectDetection lists up to 3 objects to locate in the image; see
	// AIImageDetectionRequest.ObjectDetection.
	ObjectDetection []AIImageObject
}

// DetectAIImageUpload checks whether an image, or part of it, was generated by AI, like
// DetectAIImage. It uploads the image as multipart/form-data, the format that the API
// recommends: the image is sent as it is, without the base64 encoding that makes the
// request a third larger, and the SDK holds about one copy of it.
//
// See https://docs.copyleaks.com/reference/actions/ai-image-detector/check.
func (c *Client) DetectAIImageUpload(ctx context.Context, token *AuthToken, scanID string, req *AIImageUploadRequest) (*AIImageDetectionResponse, error) {
	if err := validateID("scan ID", scanID); err != nil {
		return nil, err
	}
	if req == nil || len(req.Image) == 0 || req.Model == "" {
		return nil, invalid("Image and Model are required")
	}
	if err := validateAIImageOptions(req.Filename, req.MaskType, req.ObjectDetection); err != nil {
		return nil, err
	}
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	fields := []formField{{"filename", req.Filename}, {"model", req.Model}}
	if req.Sandbox {
		fields = append(fields, formField{"sandbox", "true"})
	}
	if req.MaskType != "" {
		fields = append(fields, formField{"maskType", req.MaskType})
	}
	if len(req.ObjectDetection) > 0 {
		objects, err := encodeJSON(req.ObjectDetection)
		if err != nil {
			return nil, fmt.Errorf("copyleaks: encode POST /v1/ai-image-detector request: %w", err)
		}
		fields = append(fields, formField{"objectDetection", string(objects)})
	}
	body, contentType, err := multipartBody("image", req.Filename, req.Image, fields)
	if err != nil {
		return nil, fmt.Errorf("copyleaks: encode POST /v1/ai-image-detector request: %w", err)
	}
	var res AIImageDetectionResponse
	err = c.do(ctx, apiRequest{
		method:      http.MethodPost,
		path:        "/v1/ai-image-detector/" + segment(scanID) + "/check",
		token:       token,
		rawBody:     body,
		contentType: contentType,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// validateAIImageOptions checks the filename, the mask type and the object detection
// list of an AI Image Detection request.
func validateAIImageOptions(filename, maskType string, objects []AIImageObject) error {
	if n := utf16Len(filename); n < 5 || n > 254 || !hasAIImageExtension(filename) {
		return invalid("Filename must be 5 to 254 characters and end with .png, .jpg, .jpeg, .bmp, .webp, .heic, .heif or .avif")
	}
	switch maskType {
	case "", AIImageMaskBinary, AIImageMaskHeatmap, AIImageMaskNone:
	default:
		return invalid(`MaskType must be "binary", "heatmap" or "none"`)
	}
	return validateAIImageObjects(objects)
}

// validateAIImageObjects checks an object detection list: up to 3 objects, each named
// with 1 to 32 characters.
func validateAIImageObjects(objects []AIImageObject) error {
	if len(objects) > maxAIImageObjects {
		return invalid(fmt.Sprintf("ObjectDetection must list at most %d objects", maxAIImageObjects))
	}
	for _, o := range objects {
		if n := utf16Len(o.Object); n < 1 || n > maxAIImageObjectLength {
			return invalid(fmt.Sprintf("each ObjectDetection object must be 1 to %d characters", maxAIImageObjectLength))
		}
	}
	return nil
}

// ResultWebhook is the webhook that receives the result of an asynchronous check, such
// as SubmitAIImageAsync and SubmitAIVideo.
type ResultWebhook struct {
	// URL is called with a POST request that carries the result (required).
	URL string `json:"url"`
	// Headers are extra headers of the webhook request, as [name, value] pairs.
	Headers [][]string `json:"headers,omitempty"`
}

// AIImageAsyncSubmission is the body of SubmitAIImageAsync.
type AIImageAsyncSubmission struct {
	// URL is the public address of the image to check (required).
	URL string `json:"url"`
	// FileName is the name of the image, ending with .png, .jpg, .jpeg, .bmp, .webp,
	// .heic, .heif or .avif, up to 254 characters (required).
	FileName string `json:"fileName"`
	// Model is the detection model, such as AIImageModel1Ultra (required).
	Model string `json:"model"`
	// MaskType is the kind of mask in the result: AIImageMaskBinary (the server default)
	// or AIImageMaskHeatmap.
	MaskType string `json:"maskType,omitempty"`
	// ObjectDetection lists up to 3 objects to locate in the image; see
	// AIImageDetectionRequest.ObjectDetection.
	ObjectDetection []AIImageObject `json:"objectDetection,omitempty"`
	// Headers are the headers that Copyleaks sends to fetch URL, as [name, value] pairs.
	Headers [][]string `json:"headers,omitempty"`
	// Webhooks is where Copyleaks sends the result (required); see ParseAIImageWebhook.
	Webhooks ResultWebhook `json:"webhooks"`
	// Sandbox runs a free test check that returns mock results.
	Sandbox bool `json:"sandbox,omitempty"`
	// DeveloperPayload is a string of up to 511 characters that Copyleaks sends back with
	// the result.
	DeveloperPayload string `json:"developerPayload,omitempty"`
}

// SubmitAIImageAsync submits the image at a URL to AI Image Detection. The check runs
// in the background, and Copyleaks posts the result to req.Webhooks.URL; decode it with
// ParseAIImageWebhook. The scan ID must be 3 to 36 characters.
//
// See https://docs.copyleaks.com/reference/actions/ai-image-detector/submit.
func (c *Client) SubmitAIImageAsync(ctx context.Context, token *AuthToken, scanID string, req *AIImageAsyncSubmission) error {
	if err := validateID("scan ID", scanID); err != nil {
		return err
	}
	if req == nil || req.URL == "" || req.Model == "" || req.Webhooks.URL == "" {
		return invalid("URL, Model and Webhooks.URL are required")
	}
	if n := utf16Len(req.FileName); n < 4 || n > 254 || !hasAIImageExtension(req.FileName) {
		return invalid("FileName must be up to 254 characters and end with .png, .jpg, .jpeg, .bmp, .webp, .heic, .heif or .avif")
	}
	if err := validateAIImageObjects(req.ObjectDetection); err != nil {
		return err
	}
	if utf16Len(req.DeveloperPayload) > maxAsyncDeveloperPayload {
		return invalid(fmt.Sprintf("DeveloperPayload must be at most %d characters", maxAsyncDeveloperPayload))
	}
	if err := VerifyAuthToken(token); err != nil {
		return err
	}
	return c.do(ctx, apiRequest{
		method:     http.MethodPut,
		path:       "/v1/ai-image-detector-async/" + segment(scanID) + "/submit",
		token:      token,
		body:       req,
		idempotent: true,
	}, nil)
}

// ParseAIImageWebhook decodes the result of SubmitAIImageAsync that Copyleaks posts to
// the webhook. It has the shape of the DetectAIImage response. When the check failed,
// the result only has Error and DeveloperPayload, so check Error first.
//
// See https://docs.copyleaks.com/reference/data-types/ai-detector/ai-image-detection-response.
func ParseAIImageWebhook(body []byte) (*AIImageDetectionResponse, error) {
	return parseWebhook[AIImageDetectionResponse]("AI image", body)
}
