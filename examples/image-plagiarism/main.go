// Command image-plagiarism searches the web for copies of an image, in sandbox mode.
//
//	export COPYLEAKS_EMAIL=you@example.com COPYLEAKS_API_KEY=your-api-key
//	go run ./examples/image-plagiarism path/to/image.jpg
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Copyleaks/copyleaks-go"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: image-plagiarism <image file>")
	}
	image, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

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

	scanID := fmt.Sprintf("image-plagiarism-%d", time.Now().Unix())
	res, err := client.CheckImagePlagiarism(ctx, token, scanID, &copyleaks.ImagePlagiarismRequest{
		Image:    image,
		Filename: filepath.Base(os.Args[1]),
		Sandbox:  true, // free test mode with mock results; remove in production
	})
	if err != nil {
		log.Fatalf("Image Plagiarism Detector: %v", err)
	}
	if res.Matches == nil {
		return
	}
	if s := res.Matches.Score; s != nil {
		fmt.Printf("%d matches: %d full, %d partial\n", s.TotalMatches, s.FullMatches, s.PartialMatches)
	}
	for _, m := range res.Matches.Internet {
		kind := "full"
		if m.MatchType == copyleaks.ImageMatchPartial {
			kind = "partial"
		}
		fmt.Printf("%s match: %s, on %d pages\n", kind, m.URL, len(m.WebPages))
	}
}
