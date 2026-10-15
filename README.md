# Copyleaks Go SDK

Copyleaks is the integrity layer for your tech stack that helps organizations and individuals understand what's human-made and original, AI-generated, or manipulated across text, image, audio and video.

This SDK gives you access to the following Copyleaks products:

* AI Text Detection
* AI Image Detection
* AI Video Detection
* Plagiarism Checker
* Image Plagiarism Detector
* Grammar Checker
* Text Moderation

## Install

```bash
go get github.com/Copyleaks/copyleaks-go
```

The SDK needs Go 1.22 or later and has no dependencies outside the standard library.

You need a Copyleaks account and its API key, which you can find on the [API Dashboard](https://api.copyleaks.com/dashboard).

## Get started: log in and detect AI text

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Copyleaks/copyleaks-go"
)

const sample = "Lions are social animals, living in groups called prides, typically consisting of several females, " +
	"their offspring, and a few males. Female lions are the primary hunters, working together to catch prey. " +
	"Lions are known for their strength, teamwork, and complex social structures."

func main() {
	ctx := context.Background()
	client, err := copyleaks.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// The token is valid for 48 hours. Reuse it, and log in again when
	// copyleaks.VerifyAuthToken(token) reports that it expired.
	token, err := client.Login(ctx, os.Getenv("COPYLEAKS_EMAIL"), os.Getenv("COPYLEAKS_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	// The text must be 255 to 100,000 characters long.
	res, err := client.DetectAIText(ctx, token, "my-first-scan", &copyleaks.AITextDetectionRequest{
		Text:    sample,
		Sandbox: true, // free test mode with mock results; remove in production
	})
	if err != nil {
		log.Fatal(err)
	}
	if res.Summary != nil {
		fmt.Printf("human: %.2f, AI: %.2f\n", res.Summary.Human, res.Summary.AI)
	}
}
```

Every method takes a `context.Context` first. Scan IDs are chosen by you and must be 3 to 36 characters long.

## AI Image Detection

```go
image, err := os.ReadFile("photo.png") // at least 512x512 pixels
if err != nil {
	log.Fatal(err)
}
res, err := client.DetectAIImageUpload(ctx, token, "my-image-scan", &copyleaks.AIImageUploadRequest{
	Image:    image,
	Filename: "photo.png",
	Model:    copyleaks.AIImageModel1Ultra,
	Sandbox:  true,
	// Optional: a confidence band for each region, and masks of objects to locate.
	MaskType:        copyleaks.AIImageMaskHeatmap,
	ObjectDetection: []copyleaks.AIImageObject{{Object: "face"}, {Object: "hand"}},
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("AI detected:", res.IsAIDetected)
if res.DetectedObjects != nil {
	for _, object := range *res.DetectedObjects {
		fmt.Println("found", object.Object)
	}
}
```

`DetectAIImageUpload` uploads the image as `multipart/form-data`, which the API recommends. `DetectAIImage`
sends the same check as base64 in JSON (`AIImageDetectionRequest.Base64`).

To check an image at a URL in the background, submit it with `SubmitAIImageAsync`. Copyleaks posts the result,
which has the shape of the `DetectAIImage` response, to your webhook. Decode it with `ParseAIImageWebhook`. When
the check fails, the webhook carries `Error` instead of a result, so check `res.Error` first.

```go
err := client.SubmitAIImageAsync(ctx, token, "my-async-image-scan", &copyleaks.AIImageAsyncSubmission{
	URL:      "https://your-server.example/images/photo.png",
	FileName: "photo.png",
	Model:    copyleaks.AIImageModel1Ultra,
	Webhooks: copyleaks.ResultWebhook{URL: "https://your-server.example/ai-image/result"},
	Sandbox:  true,
})
```

## AI Video Detection

AI Video Detection runs in the background. Submit the URL of a video, and Copyleaks posts the result to your
webhook:

```go
err := client.SubmitAIVideo(ctx, token, "my-video-scan", &copyleaks.AIVideoSubmission{
	URL:      "https://your-server.example/videos/clip.mp4",
	Filename: "clip.mp4",
	Model:    copyleaks.AIVideoModel1Pro,
	Webhooks: copyleaks.ResultWebhook{URL: "https://your-server.example/ai-video/result"},
	Sandbox:  true,
})
if err != nil {
	log.Fatal(err)
}

// In the webhook handler:
res, err := copyleaks.ParseAIVideoWebhook(body)
switch {
case err != nil:
	log.Print(err)
case res.Error != nil: // the check failed, for example because the video was too short
	fmt.Println("AI video check failed:", res.Error.Message)
case res.Summary != nil:
	fmt.Printf("AI detected: %v, overall AI ratio: %.2f\n", res.IsAIDetected, res.Summary.OverallAIRatio)
}
```

## Image Plagiarism Detector

Search the web for copies of an image, full or modified, and for the pages that show them:

```go
image, err := os.ReadFile("photo.jpg")
if err != nil {
	log.Fatal(err)
}
res, err := client.CheckImagePlagiarism(ctx, token, "my-image-plagiarism-scan", &copyleaks.ImagePlagiarismRequest{
	Image:    image,
	Filename: "photo.jpg",
	Sandbox:  true,
})
if err != nil {
	log.Fatal(err)
}
if res.Matches != nil && res.Matches.Score != nil {
	fmt.Println("matches:", res.Matches.Score.TotalMatches)
}
```

## Plagiarism Checker

A Plagiarism Checker scan runs in the background. Submit a file, an image (OCR) or a URL, and Copyleaks
sends the results to your status webhook. The scan ID may only use the characters `[a-z0-9!@$&=_()';:.,~-]`.

```go
err := client.SubmitFile(ctx, token, "my-plagiarism-scan", &copyleaks.FileSubmission{
	Content:  []byte("Text to scan."), // or Base64: the content, base64-encoded
	Filename: "document.txt",
	Properties: copyleaks.ScanProperties{
		// {STATUS} becomes completed, error, creditschecked or indexed.
		Webhooks: copyleaks.SubmitWebhooks{Status: "https://your-server.example/webhook/{STATUS}"},
		Sandbox:  true,
		// Also run AI Text Detection; the result arrives in the completed webhook.
		AIGeneratedText: &copyleaks.AIGeneratedText{Detect: true},
	},
})
if err != nil {
	log.Fatal(err)
}
```

`SubmitOCR` and `SubmitURL` work the same way. `URLSubmission.Verb` and `URLSubmission.Headers` set how
Copyleaks fetches the page. `Start`, `Delete`, `Export` and `ResendWebhook` manage submitted scans.

Set `Content` to the file as it is rather than `Base64`: the SDK then encodes it while it builds the request,
and a large upload needs about 1.4 times the file size in memory.

`ScanProperties` turns on more features of a scan, such as:

```go
copyleaks.ScanProperties{
	Webhooks: copyleaks.SubmitWebhooks{Status: "https://your-server.example/webhook/{STATUS}"},
	// AI Text Detection with AI Logic.
	AIGeneratedText: &copyleaks.AIGeneratedText{Detect: true, Explain: &copyleaks.AIGeneratedTextExplain{Enable: true}},
	// Grammar Checker, Gen-AI Overview and Citation Checker.
	WritingFeedback: &copyleaks.ScanWritingFeedback{Enable: true},
	Overview:        &copyleaks.ScanOverview{Enable: true},
	References:      &copyleaks.References{Validate: true},
	// Matches in the Copyleaks internal AI database, and masking of sensitive data.
	InternalAISourceMatch:   &copyleaks.InternalAISourceMatch{Enable: true},
	SensitiveDataProtection: &copyleaks.SensitiveDataProtection{EmailAddress: true, PhoneNumber: true},
	// Analytics IDs and the time zone of the PDF report.
	Course:       &copyleaks.Course{ID: "course-101"},
	ScanTimeZone: "America/New_York",
}
```

`Export` uploads the artifacts of a completed scan to your servers: results, the crawled version, the PDF
report, and the AI Text Detection, Grammar Checker and Gen-AI Overview reports. Decode the uploaded files with
`ParseExportedResult`, `ParseExportedCrawledVersion`, `ParseExportedAIDetection`,
`ParseExportedWritingFeedback` and `ParseExportedOverview`.

Parse the webhooks with the `Parse...Webhook` functions:

```go
http.HandleFunc("POST /webhook/completed", func(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		http.Error(w, "cannot read the body", http.StatusBadRequest)
		return
	}
	hook, err := copyleaks.ParseCompletedWebhook(body)
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if hook.Results != nil {
		fmt.Println("internet results:", len(hook.Results.Internet))
	}

	// The AI Text Detection result of the suspected-ai-text alert, or nil when there is none.
	if ai, err := hook.AIDetectionResult(); err == nil && ai != nil && ai.Summary != nil {
		fmt.Println("AI share:", ai.Summary.AI)
	}
})
```

## Grammar Checker

```go
res, err := client.CheckGrammar(ctx, token, "my-grammar-scan", &copyleaks.GrammarCheckRequest{
	Text:    "Its a text with a error.",
	Sandbox: true,
})
if err != nil {
	log.Fatal(err)
}
if res.Score != nil && res.Score.Corrections != nil {
	fmt.Println("overall score:", res.Score.Corrections.OverallScore)
}

// The names of the correction types in res.Corrections.Text.Chars.Types. This endpoint needs no login.
types, err := client.GrammarCorrectionTypes(ctx, "en")
if err != nil {
	log.Fatal(err)
}
for _, ct := range types.CorrectionTypes {
	fmt.Println(ct.ID, ct.Title)
}
```

## Text Moderation

```go
res, err := client.ModerateText(ctx, token, "my-moderation-scan", &copyleaks.TextModerationRequest{
	Text:    "Text to moderate.",
	Sandbox: true,
	Labels: []copyleaks.TextModerationLabel{
		{ID: copyleaks.TextModerationLabelToxicV1},
		{ID: copyleaks.TextModerationLabelHateSpeechV1},
	},
})
if err != nil {
	log.Fatal(err)
}
for _, label := range res.Legend {
	fmt.Println(label.Index, label.ID)
}
```

## Account and lookups

```go
credits, err := client.CreditsBalance(ctx, token)            // Plagiarism Checker credits
aiCredits, err := client.AITextCreditsBalance(ctx, token)    // AI Text Detection credits
csv, err := client.UsagesHistoryCSV(ctx, token, start, end)  // usage history
hub, err := client.RepositoryInfo(ctx, token, "my-repo-id")  // a Private Cloud Hub; needs the Admin role
```

These lookups need no login: `SupportedFileTypes`, `SupportedAITextFileTypes`, `OCRSupportedLanguages`,
`AllowedCrossLanguages` and `GrammarCorrectionTypes`.

## Errors

A request that the API rejects returns an `*copyleaks.APIError` with the status code and the response body.
Check for common cases with `errors.Is`:

```go
_, err := client.CreditsBalance(ctx, token)
switch {
case errors.Is(err, copyleaks.ErrTokenExpired):
	// Log in again.
case errors.Is(err, copyleaks.ErrUnauthorized):
	// The token or the API key was rejected.
case errors.Is(err, copyleaks.ErrRateLimited), errors.Is(err, copyleaks.ErrUnderMaintenance):
	// Retry later, or turn on WithRetry (see Options).
case errors.Is(err, copyleaks.ErrInvalidArgument):
	// A client-side check failed; no request was sent.
}
var apiErr *copyleaks.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode, string(apiErr.Body))
	// The Retry-After delay of the response, when it has one.
	fmt.Println(apiErr.RetryAfter)
}
```

A successful response larger than 16 MiB of JSON, or 64 MiB for the usage history CSV, fails with an error
that matches `copyleaks.ErrResponseTooLarge`. `WithMaxJSONResponseSize` raises the JSON limit.

## Options

```go
client, err := copyleaks.NewClient(
	copyleaks.WithRetry(copyleaks.RetryPolicy{MaxAttempts: 4}),
	copyleaks.WithTimeout(5*time.Minute),
	copyleaks.WithUserAgent("my-app/2.1"),
)
if err != nil {
	log.Fatal(err)
}
```

* `WithRetry` turns on retries, which are off by default. A call is retried after HTTP 429, HTTP 503 and connection
  errors (a timeout, or a refused, reset or dropped connection; not an invalid TLS certificate or an unknown host),
  waiting for the server's `Retry-After` delay when it sends one, and otherwise for an exponential backoff with
  jitter (`BaseDelay`, 1 second by default, doubling up to `MaxDelay`, 30 seconds by default). Only calls that are
  safe to repeat are retried: the GET calls, `Login`, the submits that carry your scan ID (`SubmitFile`,
  `SubmitOCR`, `SubmitURL`, `SubmitAIImageAsync`, `SubmitAIVideo`) and `Export`, which is not retried after HTTP
  503, as the failed export keeps its export ID. The checks that charge for each
  request (`DetectAIText`, `DetectAIImage`, `DetectAIImageUpload`, `CheckImagePlagiarism`, `CheckGrammar`,
  `ModerateText`), `Start`, `Delete` and `ResendWebhook` are never retried. The context of each call bounds the
  whole call, retries included.
* `WithTimeout` sets the timeout of each request of the default HTTP client: two minutes by default, from
  connecting to reading the whole response. Raise it to upload large files over slow links, or set 0 to rely on the
  context only.
* `WithHTTPClient` sends the requests with your own `*http.Client`, for example to use a proxy. The SDK uses it as
  it is and never changes it; `WithTimeout` does not apply to it. The default client sends the requests through
  `http.DefaultTransport`.
* `WithMaxJSONResponseSize` raises the 16 MiB limit of JSON responses, for example for the AI Image Detection masks
  of large, detailed images.
* `WithUserAgent` adds your application's token in front of the SDK's `User-Agent`.

`WithAPIBaseURL` and `WithIdentityBaseURL` change the servers. They only accept https URLs, except for
`http://localhost`, `http://127.0.0.1` and `http://[::1]` for local test servers; `NewClient` returns an
error that matches `copyleaks.ErrInvalidArgument` otherwise.

## Examples

Runnable programs for each product are in [examples](examples):

```bash
export COPYLEAKS_EMAIL=you@example.com COPYLEAKS_API_KEY=your-api-key
go run ./examples/ai-text-detection
```

## Documentation

* API reference and guides: https://docs.copyleaks.com
* Go package reference: https://pkg.go.dev/github.com/Copyleaks/copyleaks-go

## License

MIT. See [LICENSE](LICENSE).
