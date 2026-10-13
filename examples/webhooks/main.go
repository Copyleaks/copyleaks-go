// Command webhooks is a server that receives the Copyleaks webhooks of Plagiarism Checker
// scans and exports, and reads the AI Text Detection result of completed scans.
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
	"io"
	"log"
	"net/http"

	"github.com/Copyleaks/copyleaks-go"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook/completed", completed)
	mux.HandleFunc("POST /webhook/error", scanError)
	mux.HandleFunc("POST /webhook/creditschecked", creditsChecked)
	mux.HandleFunc("POST /webhook/indexed", indexed)
	mux.HandleFunc("POST /webhook/new-result", newResult)
	mux.HandleFunc("POST /export/completed", exportCompleted)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func completed(w http.ResponseWriter, r *http.Request) {
	hook, err := copyleaks.ParseCompletedWebhook(readBody(r))
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

func scanError(w http.ResponseWriter, r *http.Request) {
	hook, err := copyleaks.ParseErrorWebhook(readBody(r))
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	if hook.Error != nil {
		log.Printf("scan failed: %d %s", hook.Error.Code, hook.Error.Message)
	}
	w.WriteHeader(http.StatusOK)
}

func creditsChecked(w http.ResponseWriter, r *http.Request) {
	hook, err := copyleaks.ParseCreditsCheckedWebhook(readBody(r))
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	log.Printf("the scan will cost %d credits; start it with Client.Start", hook.Credits)
	w.WriteHeader(http.StatusOK)
}

func indexed(w http.ResponseWriter, r *http.Request) {
	if _, err := copyleaks.ParseIndexedWebhook(readBody(r)); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	log.Print("document indexed")
	w.WriteHeader(http.StatusOK)
}

func newResult(w http.ResponseWriter, r *http.Request) {
	hook, err := copyleaks.ParseNewResultWebhook(readBody(r))
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	for _, res := range hook.Internet {
		log.Printf("new result: %s (%d matched words)", res.URL, res.MatchedWords)
	}
	w.WriteHeader(http.StatusOK)
}

func exportCompleted(w http.ResponseWriter, r *http.Request) {
	hook, err := copyleaks.ParseExportCompletedWebhook(readBody(r))
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

// readBody reads up to 10 MB of the request body.
func readBody(r *http.Request) []byte {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		return nil
	}
	return body
}
