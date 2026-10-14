// Command webhooks is a server that receives the Copyleaks webhooks of Plagiarism Checker
// scans and exports, reads the AI Text Detection result of completed scans, and receives
// the results of asynchronous AI Image and AI Video Detection.
//
// Submit scans with the status webhook https://<your-server>/webhook/{STATUS} and the
// new-result webhook https://<your-server>/webhook/new-result. Copyleaks replaces
// {STATUS} with completed, error, creditschecked or indexed. In production, protect these
// endpoints, for example with a secret in SubmitWebhooks.StatusHeaders.
//
// Usage:
//
//	go run ./examples/webhooks
package main

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/Copyleaks/copyleaks-go"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook/completed", withBody(completed))
	mux.HandleFunc("POST /webhook/error", withBody(scanError))
	mux.HandleFunc("POST /webhook/creditschecked", withBody(creditsChecked))
	mux.HandleFunc("POST /webhook/indexed", withBody(indexed))
	mux.HandleFunc("POST /webhook/new-result", withBody(newResult))
	mux.HandleFunc("POST /export/completed", withBody(exportCompleted))
	mux.HandleFunc("POST /ai-image/result", withBody(aiImageResult))
	mux.HandleFunc("POST /ai-video/result", withBody(aiVideoResult))

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// maxBody is the largest webhook body that the server reads.
const maxBody = 10 << 20

// withBody reads the request body, up to maxBody, and passes it to h. It answers HTTP
// 413 to a larger body.
func withBody(h func(w http.ResponseWriter, body []byte)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		case err != nil:
			http.Error(w, "cannot read the body", http.StatusBadRequest)
		default:
			h(w, body)
		}
	}
}

func completed(w http.ResponseWriter, body []byte) {
	hook, err := copyleaks.ParseCompletedWebhook(body)
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if hook.ScannedDocument != nil && hook.Results != nil && hook.Results.Score != nil {
		log.Printf("scan %s completed: %d internet results, aggregated score %.1f%%",
			hook.ScannedDocument.ScanID, len(hook.Results.Internet), hook.Results.Score.AggregatedScore)
	}

	// The AI Text Detection result of a scan submitted with AIGeneratedText.Detect.
	ai, err := hook.AIDetectionResult()
	switch {
	case err != nil:
		log.Printf("cannot decode the AI alert, raw data: %q", hook.AIDetectionAlert().AdditionalData)
	case ai != nil && ai.Summary != nil:
		log.Printf("AI text: %.0f%% (model %s)", ai.Summary.AI*100, ai.ModelVersion)
	}
	w.WriteHeader(http.StatusOK)
}

func scanError(w http.ResponseWriter, body []byte) {
	hook, err := copyleaks.ParseErrorWebhook(body)
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if hook.Error != nil {
		log.Printf("scan failed: %d %s", hook.Error.Code, hook.Error.Message)
	}
	w.WriteHeader(http.StatusOK)
}

func creditsChecked(w http.ResponseWriter, body []byte) {
	hook, err := copyleaks.ParseCreditsCheckedWebhook(body)
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	log.Printf("the scan will cost %d credits; start it with Client.Start", hook.Credits)
	w.WriteHeader(http.StatusOK)
}

func indexed(w http.ResponseWriter, body []byte) {
	if _, err := copyleaks.ParseIndexedWebhook(body); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	log.Print("document indexed")
	w.WriteHeader(http.StatusOK)
}

func newResult(w http.ResponseWriter, body []byte) {
	hook, err := copyleaks.ParseNewResultWebhook(body)
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	for _, res := range hook.Internet {
		log.Printf("new result: %s (%d matched words)", res.URL, res.MatchedWords)
	}
	w.WriteHeader(http.StatusOK)
}

func exportCompleted(w http.ResponseWriter, body []byte) {
	hook, err := copyleaks.ParseExportCompletedWebhook(body)
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	for _, task := range hook.Tasks {
		if !task.IsHealthy || task.HTTPStatusCode < 200 || task.HTTPStatusCode > 299 {
			log.Printf("export upload to %s failed with HTTP %d", task.Endpoint, task.HTTPStatusCode)
		}
	}
	log.Printf("export completed: %v", hook.Completed)
	w.WriteHeader(http.StatusOK)
}

// aiImageResult receives the result of Client.SubmitAIImageAsync, submitted with the
// webhook https://<your-server>/ai-image/result.
func aiImageResult(w http.ResponseWriter, body []byte) {
	res, err := copyleaks.ParseAIImageWebhook(body)
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if res.Error != nil {
		log.Printf("AI image check failed: %s (%s)", res.Error.Message, res.Error.ID)
	} else if res.Summary != nil {
		log.Printf("AI image: %.0f%% AI, detected: %v", res.Summary.AI*100, res.IsAIDetected)
	}
	w.WriteHeader(http.StatusOK)
}

// aiVideoResult receives the result of Client.SubmitAIVideo, submitted with the webhook
// https://<your-server>/ai-video/result.
func aiVideoResult(w http.ResponseWriter, body []byte) {
	res, err := copyleaks.ParseAIVideoWebhook(body)
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if res.Error != nil {
		log.Printf("AI video check failed: %s (%s)", res.Error.Message, res.Error.ID)
	} else if res.Summary != nil {
		log.Printf("AI video: %.0f%% AI, detected: %v", res.Summary.OverallAIRatio*100, res.IsAIDetected)
	}
	w.WriteHeader(http.StatusOK)
}
