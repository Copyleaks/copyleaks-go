// Command text-moderation checks a text for every Text Moderation label, in sandbox mode.
//
//	export COPYLEAKS_EMAIL=you@example.com COPYLEAKS_API_KEY=your-api-key
//	go run ./examples/text-moderation
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

	var labels []copyleaks.TextModerationLabel
	for _, id := range []string{
		copyleaks.TextModerationLabelAdultV1,
		copyleaks.TextModerationLabelToxicV1,
		copyleaks.TextModerationLabelViolentV1,
		copyleaks.TextModerationLabelProfanityV1,
		copyleaks.TextModerationLabelSelfHarmV1,
		copyleaks.TextModerationLabelHarassmentV1,
		copyleaks.TextModerationLabelHateSpeechV1,
		copyleaks.TextModerationLabelDrugsV1,
		copyleaks.TextModerationLabelFirearmsV1,
		copyleaks.TextModerationLabelCybersecurityV1,
	} {
		labels = append(labels, copyleaks.TextModerationLabel{ID: id})
	}

	scanID := fmt.Sprintf("moderation-%d", time.Now().Unix())
	res, err := client.ModerateText(ctx, token, scanID, &copyleaks.TextModerationRequest{
		Text:     "This is some text to scan.",
		Sandbox:  true,
		Language: copyleaks.TextModerationLanguageEnglish,
		Labels:   labels,
	})
	if err != nil {
		log.Fatalf("Text Moderation: %v", err)
	}
	labelIDs := map[int]string{}
	for _, l := range res.Legend {
		labelIDs[l.Index] = l.ID
	}
	if res.Moderations == nil || res.Moderations.Text == nil || res.Moderations.Text.Chars == nil {
		fmt.Println("nothing was flagged")
		return
	}
	chars := res.Moderations.Text.Chars
	for i := range chars.Starts {
		if i >= len(chars.Lengths) || i >= len(chars.Labels) {
			break
		}
		fmt.Printf("%s at %d (+%d)\n", labelIDs[chars.Labels[i]], chars.Starts[i], chars.Lengths[i])
	}
}
