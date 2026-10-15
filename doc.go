// Package copyleaks is the official Go client for the Copyleaks API.
//
// It covers AI Text, Image and Video Detection, Plagiarism Checker, Image Plagiarism Detector, Grammar Checker,
// Text Moderation, account and scan management, and parsers for webhooks and exported files.
// Create one Client, log in, and pass the AuthToken to each call:
//
//	client, err := copyleaks.NewClient()
//	if err != nil {
//		return err
//	}
//	token, err := client.Login(ctx, email, apiKey)
//	if err != nil {
//		return err
//	}
//	res, err := client.DetectAIText(ctx, token, "my-scan-1",
//		&copyleaks.AITextDetectionRequest{Text: text, Sandbox: true})
//
// API reference: https://docs.copyleaks.com
package copyleaks
