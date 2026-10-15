// Command plagiarism-checker submits a text to the Plagiarism Checker, with AI Text Detection, in sandbox mode.
// The results arrive at your status webhook; see the webhooks example for a server that receives them.
//
//	export COPYLEAKS_EMAIL=you@example.com COPYLEAKS_API_KEY=your-api-key
//	export COPYLEAKS_WEBHOOK_URL='https://your-server.example/webhook/{STATUS}'
//	go run ./examples/plagiarism-checker
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Copyleaks/copyleaks-go"
)

func main() {
	webhook := os.Getenv("COPYLEAKS_WEBHOOK_URL")
	if webhook == "" {
		// example.com never receives anything; set COPYLEAKS_WEBHOOK_URL to get the results.
		webhook = "https://example.com/webhook/{STATUS}"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Retry rate-limited and failed submits; a retried submit never starts a second scan.
	client, err := copyleaks.NewClient(copyleaks.WithRetry(copyleaks.RetryPolicy{MaxAttempts: 3}))
	if err != nil {
		log.Fatal(err)
	}
	token, err := client.Login(ctx, os.Getenv("COPYLEAKS_EMAIL"), os.Getenv("COPYLEAKS_API_KEY"))
	if err != nil {
		log.Fatalf("login: %v", err)
	}

	scanID := fmt.Sprintf("plagiarism-%d", time.Now().Unix())
	err = client.SubmitFile(ctx, token, scanID, &copyleaks.FileSubmission{
		Content:  []byte("Hello world, this is a test."),
		Filename: "test.txt",
		Properties: copyleaks.ScanProperties{
			Webhooks:        copyleaks.SubmitWebhooks{Status: webhook},
			Sandbox:         true,
			AIGeneratedText: &copyleaks.AIGeneratedText{Detect: true},
		},
	})
	if err != nil {
		log.Fatalf("submit: %v", err)
	}
	fmt.Printf("scan %s submitted; the results will be sent to %s\n", scanID, webhook)

	balance, err := client.CreditsBalance(ctx, token)
	if err != nil {
		log.Fatalf("credits balance: %v", err)
	}
	fmt.Printf("remaining credits: %d\n", balance.Amount)
}
