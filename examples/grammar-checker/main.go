// Command grammar-checker checks a text with the Grammar Checker, in sandbox mode, and prints the corrections.
//
//	export COPYLEAKS_EMAIL=you@example.com COPYLEAKS_API_KEY=your-api-key
//	go run ./examples/grammar-checker
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Copyleaks/copyleaks-go"
)

const sample = "Lions are the only cat that live in groups, called pride. A prides typically consists of a few adult males, " +
	"several feales, and their offspring. Female lions, or lionesses are the primary hunters of the prid."

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

	types, err := client.GrammarCorrectionTypes(ctx, "en")
	if err != nil {
		log.Fatalf("correction types: %v", err)
	}
	typeNames := map[int]string{}
	for _, ct := range types.CorrectionTypes {
		typeNames[ct.ID] = ct.Title
	}

	scanID := fmt.Sprintf("grammar-%d", time.Now().Unix())
	res, err := client.CheckGrammar(ctx, token, scanID, &copyleaks.GrammarCheckRequest{
		Text:     sample,
		Sandbox:  true,
		Language: "en",
	})
	if err != nil {
		log.Fatalf("Grammar Checker: %v", err)
	}
	if res.Score != nil && res.Score.Corrections != nil {
		fmt.Printf("overall score: %d\n", res.Score.Corrections.OverallScore)
	}
	if res.Corrections == nil || res.Corrections.Text == nil || res.Corrections.Text.Chars == nil {
		return
	}
	chars := res.Corrections.Text.Chars
	for i := range chars.Starts {
		if i >= len(chars.Lengths) || i >= len(chars.Types) || i >= len(chars.OperationTexts) {
			break
		}
		fmt.Printf("%s at %d (+%d): %q\n", typeNames[chars.Types[i]], chars.Starts[i], chars.Lengths[i], chars.OperationTexts[i])
	}
}
