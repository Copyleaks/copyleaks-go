// Package copyleaks is the official Go client for the Copyleaks API.
//
// It covers AI Text Detection, AI Image Detection, AI Video Detection, Plagiarism
// Checker, Image Plagiarism Detector, Grammar Checker and Text Moderation, plus the
// account, scan management, Private Cloud Hub and miscellaneous endpoints, the webhook
// payloads that Copyleaks sends to your server, and the files that Export uploads.
//
// Create a Client once and reuse it. Log in to get an AuthToken, then pass the token to
// every authenticated call. Every method takes a context.Context first:
//
//	client, err := copyleaks.NewClient()
//	if err != nil {
//		return err
//	}
//	token, err := client.Login(ctx, email, apiKey)
//	if err != nil {
//		return err
//	}
//	res, err := client.DetectAIText(ctx, token, "my-scan-1", &copyleaks.AITextDetectionRequest{
//		Text:    text,
//		Sandbox: true,
//	})
//
// Requests that the API rejects return an *APIError. Use errors.Is with ErrUnauthorized,
// ErrRateLimited or ErrUnderMaintenance to check for those cases. Arguments that fail the
// client-side checks return an error that matches ErrInvalidArgument, before any request
// is sent. The package retries nothing unless WithRetry turns retries on, never panics on
// bad input and never writes to standard output or standard error.
//
// API reference: https://docs.copyleaks.com
package copyleaks
