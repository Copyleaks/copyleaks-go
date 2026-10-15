// Command ai-image-detection checks whether an image (at least 512x512 pixels) is AI-generated, in sandbox mode.
//
//	export COPYLEAKS_EMAIL=you@example.com COPYLEAKS_API_KEY=your-api-key
//	go run ./examples/ai-image-detection path/to/image.png
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
		log.Fatal("usage: ai-image-detection <image file>")
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

	scanID := fmt.Sprintf("ai-image-%d", time.Now().Unix())
	// DetectAIImageUpload sends the image as multipart/form-data; DetectAIImage sends it as base64 in JSON.
	res, err := client.DetectAIImageUpload(ctx, token, scanID, &copyleaks.AIImageUploadRequest{
		Image:    image,
		Filename: filepath.Base(os.Args[1]),
		Model:    copyleaks.AIImageModel1Ultra,
		Sandbox:  true,
		MaskType: copyleaks.AIImageMaskHeatmap, // a confidence band for each AI-generated region
	})
	if err != nil {
		log.Fatalf("AI Image Detection: %v", err)
	}
	if res.Summary != nil {
		fmt.Printf("AI detected: %v (human: %.0f%%, AI: %.0f%%)\n", res.IsAIDetected, res.Summary.Human*100, res.Summary.AI*100)
	}
}
