package copyleaks

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// The webhook fixtures in testdata are the examples of the API reference
// (https://docs.copyleaks.com/reference/data-types/authenticity/webhooks).

func TestParseCompletedWebhook(t *testing.T) {
	w, err := ParseCompletedWebhook([]byte(fixture(t, "webhook_completed.json")))
	if err != nil {
		t.Fatal(err)
	}
	if w.Status != 0 || w.DeveloperPayload != "Custom developer payload" {
		t.Errorf("status %d, payload %q", w.Status, w.DeveloperPayload)
	}
	doc := w.ScannedDocument
	if doc.ScanID != "string" || doc.Metadata == nil || doc.Metadata.FinalURL != "string" || doc.Metadata.LastModificationDate != "string" {
		t.Errorf("scannedDocument: %+v %+v", doc, doc.Metadata)
	}
	r := w.Results
	if len(r.Internet) != 1 || r.Internet[0].URL != "string" || r.Internet[0].Metadata.CanonicalURL != "string" {
		t.Errorf("internet: %+v", r.Internet)
	}
	if len(r.Database) != 1 || r.Database[0].ScanID != "string" || len(r.Batch) != 1 || r.Batch[0].ID != "string" {
		t.Errorf("database %+v, batch %+v", r.Database, r.Batch)
	}
	if len(r.Repositories) != 1 || r.Repositories[0].RepositoryID != "string" || r.Repositories[0].Metadata.SubmittedBy != "string" {
		t.Errorf("repositories: %+v", r.Repositories)
	}
	if r.Score == nil || *r.Score != (ResultsScore{}) {
		t.Errorf("score: %+v", r.Score)
	}
	want := []Alert{{Category: 2, Code: "string", Title: "string", Message: "string", HelpLink: "string", Severity: 0, AdditionalData: "string"}}
	if !reflect.DeepEqual(w.Notifications.Alerts, want) {
		t.Errorf("alerts: %+v", w.Notifications.Alerts)
	}
	if alert := w.AIDetectionAlert(); alert != nil {
		t.Errorf("AIDetectionAlert() = %+v, want nil", alert)
	}
}

func TestParseCompletedWebhookScore(t *testing.T) {
	body := `{"status":0,"results":{"score":{"identicalWords":10,"minorChangedWords":2,"relatedMeaningWords":3,"aggregatedScore":12.5}}}`
	w, err := ParseCompletedWebhook([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if want := (ResultsScore{IdenticalWords: 10, MinorChangedWords: 2, RelatedMeaningWords: 3, AggregatedScore: 12.5}); *w.Results.Score != want {
		t.Errorf("score = %+v, want %+v", *w.Results.Score, want)
	}
}

func TestParseErrorWebhook(t *testing.T) {
	w, err := ParseErrorWebhook([]byte(fixture(t, "webhook_error.json")))
	if err != nil {
		t.Fatal(err)
	}
	want := &ErrorWebhook{
		Status:           1,
		DeveloperPayload: "Custom developer payload",
		Error: &WebhookError{
			Type:    "invalid_request_error",
			ID:      "missing_parameter",
			Code:    1,
			Message: "Bad request. One or several required parameters are missing or incorrect.",
		},
	}
	if !reflect.DeepEqual(w, want) {
		t.Errorf("got %+v %+v, want %+v %+v", w, w.Error, want, want.Error)
	}

	details := `{"status":1,"error":{"code":1,"details":[{"param":"scanId","message":"too short"}],"url":"https://docs.copyleaks.com"}}`
	w, err = ParseErrorWebhook([]byte(details))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.Error.Details, []WebhookErrorDetail{{Param: "scanId", Message: "too short"}}) || w.Error.URL != "https://docs.copyleaks.com" {
		t.Errorf("error: %+v", w.Error)
	}
}

func TestParseCreditsCheckedWebhook(t *testing.T) {
	w, err := ParseCreditsCheckedWebhook([]byte(fixture(t, "webhook_credits_checked.json")))
	if err != nil {
		t.Fatal(err)
	}
	if want := (&CreditsCheckedWebhook{Status: 2, DeveloperPayload: "Custom developer payload", Credits: 1}); !reflect.DeepEqual(w, want) {
		t.Errorf("got %+v, want %+v", w, want)
	}
}

func TestParseIndexedWebhook(t *testing.T) {
	w, err := ParseIndexedWebhook([]byte(fixture(t, "webhook_indexed.json")))
	if err != nil {
		t.Fatal(err)
	}
	if want := (&IndexedWebhook{Status: 3, DeveloperPayload: "Custom developer payload"}); !reflect.DeepEqual(w, want) {
		t.Errorf("got %+v, want %+v", w, want)
	}
}

func TestParseExportCompletedWebhook(t *testing.T) {
	w, err := ParseExportCompletedWebhook([]byte(fixture(t, "webhook_export_completed.json")))
	if err != nil {
		t.Fatal(err)
	}
	if !w.Completed || w.DeveloperPayload != "This is my payload" || len(w.Tasks) != 3 {
		t.Fatalf("got %+v", w)
	}
	want := ExportTask{Endpoint: "https://yourserver.com/export/export-id/pdf-report", IsHealthy: true, HTTPStatusCode: 200}
	if w.Tasks[1] != want {
		t.Errorf("task = %+v, want %+v", w.Tasks[1], want)
	}
}

func TestParseNewResultWebhook(t *testing.T) {
	// The API reference shows score as a number.
	w, err := ParseNewResultWebhook([]byte(fixture(t, "webhook_new_result.json")))
	if err != nil {
		t.Fatal(err)
	}
	if w.DeveloperPayload != "string" || w.Score == nil || len(w.Internet) != 1 || len(w.Database) != 1 || len(w.Batch) != 1 || len(w.Repositories) != 1 {
		t.Fatalf("got %+v", w)
	}
	if w.Internet[0].MatchedWords != 0 || w.Repositories[0].Metadata.SubmittedBy != "string" {
		t.Errorf("results: %+v %+v", w.Internet[0], w.Repositories[0])
	}

	// The server sends score as an object.
	w, err = ParseNewResultWebhook([]byte(`{"score":{"identicalWords":4,"aggregatedScore":7.5},"internet":[{"id":"r1","matchedWords":4,"tags":[{"code":"c","title":"t","description":"d"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if *w.Score != (ResultsScore{IdenticalWords: 4, AggregatedScore: 7.5}) {
		t.Errorf("score = %+v", *w.Score)
	}
	if want := []ResultTag{{Code: "c", Title: "t", Description: "d"}}; !reflect.DeepEqual(w.Internet[0].Tags, want) {
		t.Errorf("tags = %+v", w.Internet[0].Tags)
	}

	w, err = ParseNewResultWebhook([]byte(`{"score":42.5}`))
	if err != nil {
		t.Fatal(err)
	}
	if w.Score.AggregatedScore != 42.5 {
		t.Errorf("numeric score = %+v", *w.Score)
	}
}

func TestParseWebhookErrors(t *testing.T) {
	if _, err := ParseCompletedWebhook([]byte(`{"status":`)); err == nil || !strings.HasPrefix(err.Error(), "copyleaks: parse completed webhook: ") {
		t.Errorf("err = %v", err)
	}
	if _, err := ParseNewResultWebhook([]byte(`{"score":"high"}`)); err == nil {
		t.Error("a string score must fail")
	}
}

// completedWithAlerts returns a completed webhook body with the given alerts.
func completedWithAlerts(t *testing.T, alerts ...any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{"status": 0, "notifications": map[string]any{"alerts": alerts}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// aiAlert returns a suspected-ai-text alert with the given additional data.
func aiAlert(additionalData string) map[string]any {
	return map[string]any{
		"category":       2,
		"code":           AlertCodeSuspectedAIText,
		"title":          "Suspected Cheating: AI Text Detected",
		"message":        "We are unable to verify that the text was written by a human.",
		"helpLink":       "https://docs.copyleaks.com",
		"severity":       4,
		"additionalData": additionalData,
	}
}

func TestCompletedWebhookAIDetectionResult(t *testing.T) {
	aiJSON := fixture(t, "ai_text_response.json")
	other := map[string]any{"category": 3, "code": "suspected-cheating-detected", "additionalData": aiJSON}

	w, err := ParseCompletedWebhook(completedWithAlerts(t, nil, other, aiAlert(aiJSON+"\x00\x00\n\t "), aiAlert("{}")))
	if err != nil {
		t.Fatal(err)
	}
	alert := w.AIDetectionAlert()
	if alert == nil || alert.Code != AlertCodeSuspectedAIText || alert.Severity != 4 || alert.HelpLink != "https://docs.copyleaks.com" {
		t.Fatalf("AIDetectionAlert() = %+v", alert)
	}
	if !strings.HasSuffix(alert.AdditionalData, "\x00\x00\n\t ") {
		t.Error("AdditionalData must keep the raw string")
	}
	res, err := w.AIDetectionResult()
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.ModelVersion != "v11.0" || res.Summary.AI != 1 || res.Explain == nil || res.ScannedDocument.ScanID != "scan-id" {
		t.Fatalf("AIDetectionResult() = %+v", res)
	}
	if fromAlert, err := alert.AIDetectionResult(); err != nil || !reflect.DeepEqual(fromAlert, res) {
		t.Errorf("alert.AIDetectionResult() = %+v, %v", fromAlert, err)
	}
}

func TestAIDetectionResultEmptyCases(t *testing.T) {
	tests := map[string][]byte{
		"no notifications":    []byte(`{"status":0}`),
		"no alerts":           completedWithAlerts(t),
		"other alert only":    completedWithAlerts(t, map[string]any{"code": "suspected-white-text", "additionalData": `{"summary":{"ai":1}}`}),
		"empty data":          completedWithAlerts(t, aiAlert("")),
		"only padding":        completedWithAlerts(t, aiAlert("\x00\n \t")),
		"null":                completedWithAlerts(t, aiAlert("null")),
		"array":               completedWithAlerts(t, aiAlert("[1,2]")),
		"string":              completedWithAlerts(t, aiAlert(`"text"`)),
		"null alert entry":    completedWithAlerts(t, nil),
		"missing data field":  completedWithAlerts(t, map[string]any{"code": AlertCodeSuspectedAIText}),
		"null data with nuls": completedWithAlerts(t, aiAlert("null\x00\x00")),
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			w, err := ParseCompletedWebhook(body)
			if err != nil {
				t.Fatal(err)
			}
			res, err := w.AIDetectionResult()
			if res != nil || err != nil {
				t.Errorf("AIDetectionResult() = %+v, %v; want nil, nil", res, err)
			}
		})
	}
}

func TestAIDetectionResultInvalidData(t *testing.T) {
	for name, data := range map[string]string{
		"malformed JSON":  `{"summary":`,
		"trailing text":   `{"summary":{"ai":1}} extra`,
		"wrong type":      `{"summary":5}`,
		"bool modelVer":   `{"modelVersion":true}`,
		"leading garbage": `x{"summary":{"ai":1}}`,
	} {
		t.Run(name, func(t *testing.T) {
			w, err := ParseCompletedWebhook(completedWithAlerts(t, aiAlert(data)))
			if err != nil {
				t.Fatal(err)
			}
			res, err := w.AIDetectionResult()
			if res != nil || err == nil || !strings.HasPrefix(err.Error(), "copyleaks: decode suspected-ai-text additionalData: ") {
				t.Errorf("AIDetectionResult() = %+v, %v; want an error", res, err)
			}
			if w.AIDetectionAlert().AdditionalData != data {
				t.Error("AdditionalData must keep the raw string")
			}
		})
	}
}

func TestAIDetectionResultOnNilValues(t *testing.T) {
	var w *CompletedWebhook
	if w.AIDetectionAlert() != nil {
		t.Error("nil webhook: alert must be nil")
	}
	if res, err := w.AIDetectionResult(); res != nil || err != nil {
		t.Errorf("nil webhook: %v, %v", res, err)
	}
	var a *Alert
	if res, err := a.AIDetectionResult(); res != nil || err != nil {
		t.Errorf("nil alert: %v, %v", res, err)
	}
	other := &Alert{Code: "suspected-white-text", AdditionalData: `{"summary":{"ai":1}}`}
	if res, err := other.AIDetectionResult(); res != nil || err != nil {
		t.Errorf("other alert: %v, %v", res, err)
	}
}

func TestAITextDetectionResponseNumericModelVersion(t *testing.T) {
	for raw, want := range map[string]string{
		`{"modelVersion":"v11.0"}`: "v11.0",
		`{"modelVersion":11}`:      "11",
		`{"modelVersion":11.5}`:    "11.5",
		`{"modelVersion":null}`:    "",
		`{}`:                       "",
	} {
		var res AITextDetectionResponse
		if err := json.Unmarshal([]byte(raw), &res); err != nil {
			t.Errorf("%s: %v", raw, err)
			continue
		}
		if res.ModelVersion != want {
			t.Errorf("%s: ModelVersion = %q, want %q", raw, res.ModelVersion, want)
		}
	}
	var res AITextDetectionResponse
	if err := json.Unmarshal([]byte(`{"modelVersion":11,"summary":{"human":0.25,"ai":0.75},"results":null}`), &res); err != nil {
		t.Fatal(err)
	}
	if res.Summary.AI != 0.75 || res.Results != nil {
		t.Errorf("other fields: %+v", res)
	}
}

func TestStartedScanAcceptsStrings(t *testing.T) {
	var res StartResponse
	if err := json.Unmarshal([]byte(`{"success":["scan-1"],"failed":[{"id":"scan-2"}]}`), &res); err != nil {
		t.Fatal(err)
	}
	want := StartResponse{Success: []StartedScan{{ID: "scan-1"}}, Failed: []StartedScan{{ID: "scan-2"}}}
	if !reflect.DeepEqual(res, want) {
		t.Errorf("got %+v, want %+v", res, want)
	}
}
