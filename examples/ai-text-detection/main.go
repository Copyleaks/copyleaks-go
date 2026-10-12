// Command ai-text-detection logs in and checks a text with AI Text Detection, in sandbox
// mode, which is free and returns mock results.
//
// Usage:
//
//	export COPYLEAKS_EMAIL=you@example.com COPYLEAKS_API_KEY=your-api-key
//	go run ./examples/ai-text-detection
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Copyleaks/copyleaks-go"
)

const sample = "Lions are social animals, living in groups called prides, typically consisting of several females, " +
	"their offspring, and a few males. Female lions are the primary hunters, working together to catch prey. " +
	"Lions are known for their strength, teamwork, and complex social structures."

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := copyleaks.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	token, err := client.Login(ctx, os.Getenv("COPYLEAKS_EMAIL"), os.Getenv("COPYLEAKS_API_KEY"))
	if err != nil {
		log.Fatalf("login: %v", err)
	}

	scanID := fmt.Sprintf("ai-text-%d", time.Now().Unix())
	res, err := client.DetectAIText(ctx, token, scanID, &copyleaks.AITextDetectionRequest{
		Text:    sample,
		Sandbox: true,
	})
	if err != nil {
		log.Fatalf("AI Text Detection: %v", err)
	}
	if res.Summary != nil {
		fmt.Printf("human: %.0f%%, AI: %.0f%% (model %s)\n", res.Summary.Human*100, res.Summary.AI*100, res.ModelVersion)
	}
	for _, section := range res.Results {
		if section.Classification != 2 {
			continue
		}
		for _, match := range section.Matches {
			if match.Text != nil && match.Text.Chars != nil {
				fmt.Printf("AI-generated text at character offsets %v, lengths %v\n", match.Text.Chars.Starts, match.Text.Chars.Lengths)
			}
		}
	}
}
