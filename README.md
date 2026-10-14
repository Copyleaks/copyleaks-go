# Copyleaks Go SDK

Copyleaks is the integrity layer for your tech stack that helps organizations and individuals understand what's human-made and original, AI-generated, or manipulated across text, image, audio and video.

This SDK gives you access to the following Copyleaks products:

* AI Text Detection
* AI Image Detection
* Plagiarism Checker
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
res, err := client.DetectAIImage(ctx, token, "my-image-scan", &copyleaks.AIImageDetectionRequest{
	Base64:   base64.StdEncoding.EncodeToString(image),
	Filename: "photo.png",
	Model:    copyleaks.AIImageModel1Ultra,
	Sandbox:  true,
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("AI detected:", res.IsAIDetected)
```

## Plagiarism Checker

A Plagiarism Checker scan runs in the background. Submit a file, an image (OCR) or a URL, and Copyleaks
sends the results to your status webhook. The scan ID may only use the characters `[a-z0-9!@$&=_()';:.,~-]`.

```go
err := client.SubmitFile(ctx, token, "my-plagiarism-scan", &copyleaks.FileSubmission{
	Base64:   base64.StdEncoding.EncodeToString([]byte("Text to scan.")),
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

`SubmitOCR` and `SubmitURL` work the same way. `Start`, `Delete`, `Export` and `ResendWebhook` manage
submitted scans, and `CreditsBalance` and `UsagesHistoryCSV` report on your account.

Parse the webhooks with the `Parse...Webhook` functions:

```go
http.HandleFunc("POST /webhook/completed", func(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
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
	// Retry later. The SDK never retries by itself.
case errors.Is(err, copyleaks.ErrInvalidArgument):
	// A client-side check failed; no request was sent.
}
var apiErr *copyleaks.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode, string(apiErr.Body))
}
```

A successful response larger than 16 MiB of JSON, or 64 MiB for the usage history CSV, fails with an error
that matches `copyleaks.ErrResponseTooLarge`.

## Options

```go
client, err := copyleaks.NewClient(
	copyleaks.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	copyleaks.WithUserAgent("my-app/2.1"),
)
if err != nil {
	log.Fatal(err)
}
```

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
