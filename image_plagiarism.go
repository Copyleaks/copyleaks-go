package copyleaks

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Copyleaks/copyleaks-go/internal/upload"
	"github.com/Copyleaks/copyleaks-go/internal/validate"
)

// imagePlagiarismExtensions are the file extensions that the Image Plagiarism Detector
// accepts.
var imagePlagiarismExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".ico"}

// ImagePlagiarismRequest is the request of CheckImagePlagiarism.
type ImagePlagiarismRequest struct {
	// Image is the image file (required): under 20 MB and at most 75 megapixels, in JPEG,
	// PNG, GIF, BMP, WebP or ICO format. The server checks the size.
	Image []byte
	// Filename is the name of the image, up to 255 characters, ending with .jpg, .jpeg,
	// .png, .gif, .bmp, .webp or .ico (required).
	Filename string
	// Sandbox runs a free test check that returns mock results.
	Sandbox bool
	// DeveloperPayload is sent back as it is in ImagePlagiarismResponse.DeveloperPayload.
	DeveloperPayload string
}

// The values of ImagePlagiarismMatch.MatchType.
const (
	// ImageMatchFull is an exact or near-exact copy of the image.
	ImageMatchFull = 0
	// ImageMatchPartial is a cropped, resized, recolored or otherwise modified copy.
	ImageMatchPartial = 1
)

// ImagePlagiarismResponse is the result of the Image Plagiarism Detector.
//
// See https://docs.copyleaks.com/reference/data-types/authenticity/results/image-plagiarism-response.
type ImagePlagiarismResponse struct {
	// DeveloperPayload is the DeveloperPayload of the request.
	DeveloperPayload string `json:"developerPayload"`
	// ScannedImage is information about the image and the check.
	ScannedImage *ScannedImage `json:"scannedImage"`
	// Matches are the copies of the image found on the web.
	Matches *ImagePlagiarismMatches `json:"matches"`
}

// ScannedImage is information about the image of an Image Plagiarism Detector check.
type ScannedImage struct {
	// ScanID is the scan ID given by you.
	ScanID string `json:"scanId"`
	// ExpectedCredits is the number of credits that the check was expected to use.
	ExpectedCredits int `json:"expectedCredits"`
	// ActualCredits is the number of credits that the check used.
	ActualCredits int `json:"actualCredits"`
	// CreationTime is when the check was created, in ISO 8601 format (UTC).
	CreationTime string `json:"creationTime"`
	// Width is the width of the image, in pixels.
	Width int `json:"width"`
	// Height is the height of the image, in pixels.
	Height int `json:"height"`
	// Filename is the Filename of the request.
	Filename string `json:"filename"`
}

// ImagePlagiarismMatches are the copies of an image found on the web.
type ImagePlagiarismMatches struct {
	// Internet are the matching images, full matches first, then partial matches. Each
	// image URL appears once. No matches and a zero Score mean that no copy was found.
	Internet []ImagePlagiarismMatch `json:"internet"`
	// Score counts the matches.
	Score *ImagePlagiarismScore `json:"score"`
}

// ImagePlagiarismMatch is a copy of an image found on the web.
type ImagePlagiarismMatch struct {
	// URL is the address of the matching image.
	URL string `json:"url"`
	// MatchType is ImageMatchFull or ImageMatchPartial.
	MatchType int `json:"matchType"`
	// WebPages are the pages that show the image. It is empty when no page was found.
	WebPages []ImagePlagiarismWebPage `json:"webPages"`
}

// ImagePlagiarismWebPage is a web page that shows a matching image.
type ImagePlagiarismWebPage struct {
	// URL is the address of the page.
	URL string `json:"url"`
}

// ImagePlagiarismScore counts the matches of an Image Plagiarism Detector check.
type ImagePlagiarismScore struct {
	// TotalMatches is the number of matching images.
	TotalMatches int `json:"totalMatches"`
	// FullMatches is the number of full matches.
	FullMatches int `json:"fullMatches"`
	// PartialMatches is the number of partial matches.
	PartialMatches int `json:"partialMatches"`
}

// CheckImagePlagiarism searches the web for copies of an image, full or modified, and
// for the pages that show them. The image is uploaded as multipart/form-data, as the
// endpoint requires. The scan ID must be 3 to 36 characters.
//
// See https://docs.copyleaks.com/reference/actions/image-plagiarism-detector/check.
func (c *Client) CheckImagePlagiarism(ctx context.Context, token *AuthToken, scanID string, req *ImagePlagiarismRequest) (*ImagePlagiarismResponse, error) {
	if err := validate.ID("scan ID", scanID); err != nil {
		return nil, invalid(err.Error())
	}
	if req == nil || len(req.Image) == 0 {
		return nil, invalid("Image is required")
	}
	if n := validate.UTF16Len(req.Filename); n < 1 || n > 255 || !hasExtension(req.Filename, imagePlagiarismExtensions) {
		return nil, invalid("Filename must be up to 255 characters and end with .jpg, .jpeg, .png, .gif, .bmp, .webp or .ico")
	}
	if err := VerifyAuthToken(token); err != nil {
		return nil, err
	}
	fields := []upload.Field{{Name: "filename", Value: req.Filename}}
	if req.Sandbox {
		fields = append(fields, upload.Field{Name: "sandbox", Value: "true"})
	}
	if req.DeveloperPayload != "" {
		fields = append(fields, upload.Field{Name: "developerPayload", Value: req.DeveloperPayload})
	}
	body, contentType, err := upload.MultipartBody("image", req.Filename, req.Image, fields)
	if err != nil {
		return nil, fmt.Errorf("copyleaks: encode POST /v1/image-plagiarism-detector request: %w", err)
	}
	var res ImagePlagiarismResponse
	err = c.do(ctx, apiRequest{
		method:      http.MethodPost,
		path:        "/v1/image-plagiarism-detector/" + segment(scanID) + "/check",
		token:       token,
		rawBody:     body,
		contentType: contentType,
	}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
