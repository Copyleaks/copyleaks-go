package copyleaks

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"
)

// endpointCase describes one SDK method: the request it must send, the response the
// test server answers with, and a check of the decoded result. server is "api" or
// "identity", auth tells whether the request carries the token, and body is the
// expected JSON request body, or empty when the request has none.
type endpointCase struct {
	name     string
	server   string
	method   string
	path     string
	query    url.Values
	auth     bool
	body     string
	status   int
	response string
	call     func(ctx context.Context, c *Client) (any, error)
	check    func(t *testing.T, got any)
}

// endpointCases covers every endpoint of the SDK.
func endpointCases(t *testing.T) []endpointCase {
	tok := testToken()
	props := ScanProperties{
		Webhooks: SubmitWebhooks{Status: "https://example.com/webhook/{STATUS}"},
		Sandbox:  true,
	}
	propsJSON := `{"webhooks":{"status":"https://example.com/webhook/{STATUS}"},"sandbox":true}`

	return []endpointCase{
		{
			name:     "Login",
			server:   "identity",
			method:   http.MethodPost,
			path:     "/v3/account/login/api",
			body:     `{"email":"user@example.com","key":"api-key"}`,
			status:   http.StatusOK,
			response: `{"access_token":"abc",".issued":"2026-10-12T10:00:00.1234567+03:00",".expires":"2026-10-14T10:00:00.1234567+03:00"}`,
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Login(ctx, "user@example.com", "api-key")
			},
			check: func(t *testing.T, got any) {
				token := got.(*AuthToken)
				wantExpires := time.Date(2026, 10, 14, 7, 0, 0, 123456700, time.UTC)
				if token.AccessToken != "abc" || !token.Expires.Equal(wantExpires) {
					t.Errorf("token = %q expiring %v, want %q expiring %v", token.AccessToken, token.Expires, "abc", wantExpires)
				}
			},
		},
		{
			name:     "SubmitFile",
			method:   http.MethodPut,
			path:     "/v3/scans/submit/file/scan-1",
			auth:     true,
			body:     `{"base64":"aGVsbG8gd29ybGQ=","filename":"file.txt","properties":` + propsJSON + `}`,
			status:   http.StatusCreated,
			response: "",
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Base64: "aGVsbG8gd29ybGQ=", Filename: "file.txt", Properties: props})
			},
		},
		{
			name:     "SubmitOCR",
			method:   http.MethodPut,
			path:     "/v3/scans/submit/ocr/scan-1",
			auth:     true,
			body:     `{"base64":"iVBORw0KGgo=","filename":"image.png","langCode":"en","properties":` + propsJSON + `}`,
			status:   http.StatusCreated,
			response: "",
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.SubmitOCR(ctx, tok, "scan-1", &OCRSubmission{Base64: "iVBORw0KGgo=", Filename: "image.png", LangCode: "en", Properties: props})
			},
		},
		{
			name:     "SubmitURL",
			method:   http.MethodPut,
			path:     "/v3/scans/submit/url/scan-1",
			auth:     true,
			body:     `{"url":"https://example.com/page","properties":` + propsJSON + `}`,
			status:   http.StatusCreated,
			response: "",
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.SubmitURL(ctx, tok, "scan-1", &URLSubmission{URL: "https://example.com/page", Properties: props})
			},
		},
		{
			name:   "Export",
			method: http.MethodPost,
			path:   "/v3/downloads/scan-1/export/export-1",
			auth:   true,
			body: `{"completionWebhook":"https://example.com/export/done","maxRetries":3,"developerPayload":"payload",` +
				`"results":[{"id":"r1","endpoint":"https://example.com/r1","verb":"POST","headers":[["k","v"]]}],` +
				`"pdfReport":{"endpoint":"https://example.com/pdf","verb":"PUT"},` +
				`"crawledVersion":{"endpoint":"https://example.com/crawled","verb":"POST"}}`,
			status:   http.StatusNoContent,
			response: "",
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Export(ctx, tok, "scan-1", "export-1", &ExportRequest{
					CompletionWebhook: "https://example.com/export/done",
					MaxRetries:        3,
					DeveloperPayload:  "payload",
					Results:           []ExportResult{{ID: "r1", Endpoint: "https://example.com/r1", Verb: "POST", Headers: [][]string{{"k", "v"}}}},
					PDFReport:         &ExportTarget{Endpoint: "https://example.com/pdf", Verb: "PUT"},
					CrawledVersion:    &ExportTarget{Endpoint: "https://example.com/crawled", Verb: "POST"},
				})
			},
		},
		{
			name:     "Start",
			method:   http.MethodPatch,
			path:     "/v3/scans/start",
			auth:     true,
			body:     `{"trigger":["scan-1","scan-2"],"errorHandling":1}`,
			status:   http.StatusOK,
			response: `{"success":[{"id":"scan-1"}],"failed":[{"id":"scan-2"}]}`,
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Start(ctx, tok, &StartRequest{Trigger: []string{"scan-1", "scan-2"}, ErrorHandling: StartErrorHandlingIgnore})
			},
			check: func(t *testing.T, got any) {
				want := &StartResponse{Success: []StartedScan{{ID: "scan-1"}}, Failed: []StartedScan{{ID: "scan-2"}}}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("got %+v, want %+v", got, want)
				}
			},
		},
		{
			name:     "Delete",
			method:   http.MethodPatch,
			path:     "/v3.1/scans/delete",
			auth:     true,
			body:     `{"scans":[{"id":"scan-1"},{"id":"scan-2"}],"purge":true,"completionWebhook":"https://example.com/deleted"}`,
			status:   http.StatusAccepted,
			response: "{}",
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Delete(ctx, tok, &DeleteRequest{
					Scans:             []DeleteScan{{ID: "scan-1"}, {ID: "scan-2"}},
					Purge:             true,
					CompletionWebhook: "https://example.com/deleted",
				})
			},
		},
		{
			name:     "ResendWebhook",
			method:   http.MethodPost,
			path:     "/v3/scans/scan-1/webhooks/resend",
			auth:     true,
			status:   http.StatusAccepted,
			response: "{}",
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.ResendWebhook(ctx, tok, "scan-1")
			},
		},
		{
			name:     "CreditsBalance",
			method:   http.MethodGet,
			path:     "/v3/scans/credits",
			auth:     true,
			status:   http.StatusOK,
			response: `{"Amount":100}`,
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.CreditsBalance(ctx, tok)
			},
			check: func(t *testing.T, got any) {
				if got := got.(*CreditsBalance).Amount; got != 100 {
					t.Errorf("Amount = %d, want 100", got)
				}
			},
		},
		{
			name:     "UsagesHistoryCSV",
			method:   http.MethodGet,
			path:     "/v3/scans/usages/history",
			query:    url.Values{"start": {"01-01-2024"}, "end": {"31-01-2024"}},
			auth:     true,
			status:   http.StatusOK,
			response: "Date,Credits\n01-01-2024,5\n",
			call: func(ctx context.Context, c *Client) (any, error) {
				start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
				return c.UsagesHistoryCSV(ctx, tok, start, end)
			},
			check: func(t *testing.T, got any) {
				if got := string(got.([]byte)); got != "Date,Credits\n01-01-2024,5\n" {
					t.Errorf("CSV = %q", got)
				}
			},
		},
		{
			name:     "ReleaseNotes",
			method:   http.MethodGet,
			path:     "/v3/release-logs.json",
			status:   http.StatusOK,
			response: fixture(t, "release_notes.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.ReleaseNotes(ctx)
			},
			check: func(t *testing.T, got any) {
				notes := got.(*ReleaseNotes)
				if len(notes.Updates) != 2 || notes.Updates[0].MsgType != "NEW" || notes.Updates[0].Date != "2023-12-07T09:00:00Z" || notes.Updates[0].HTMLDescription == "" {
					t.Errorf("got %+v", notes)
				}
			},
		},
		{
			name:     "SupportedFileTypes",
			method:   http.MethodGet,
			path:     "/v3/miscellaneous/supported-file-types",
			status:   http.StatusOK,
			response: fixture(t, "supported_file_types.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.SupportedFileTypes(ctx)
			},
			check: func(t *testing.T, got any) {
				types := got.(*SupportedFileTypes)
				if len(types.Textual) != 45 || types.Textual[0] != "pdf" || len(types.OCR) != 7 || types.OCR[1] != "png" {
					t.Errorf("got %+v", types)
				}
			},
		},
		{
			name:     "OCRSupportedLanguages",
			method:   http.MethodGet,
			path:     "/v3/miscellaneous/ocr-languages-list",
			status:   http.StatusOK,
			response: `["af","sq","en"]`,
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.OCRSupportedLanguages(ctx)
			},
			check: func(t *testing.T, got any) {
				if want := []string{"af", "sq", "en"}; !reflect.DeepEqual(got, want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:     "DetectAIText",
			method:   http.MethodPost,
			path:     "/v2/writer-detector/scan-1/check",
			auth:     true,
			body:     `{"text":"` + lionsText + `","sandbox":true,"language":"en"}`,
			status:   http.StatusOK,
			response: fixture(t, "ai_text_response.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.DetectAIText(ctx, tok, "scan-1", &AITextDetectionRequest{Text: lionsText, Sandbox: true, Language: "en"})
			},
			check: func(t *testing.T, got any) {
				res := got.(*AITextDetectionResponse)
				if res.ModelVersion != "v11.0" || res.Summary == nil || res.Summary.AI != 1 || res.Summary.Human != 0 {
					t.Errorf("model or summary: %+v %+v", res.ModelVersion, res.Summary)
				}
				if len(res.Results) != 1 || res.Results[0].Classification != 2 || res.Results[0].Probability != 0.7316979 {
					t.Fatalf("results: %+v", res.Results)
				}
				chars := res.Results[0].Matches[0].Text.Chars
				if !reflect.DeepEqual(chars.Starts, []int{0}) || !reflect.DeepEqual(chars.Lengths, []int{554}) {
					t.Errorf("chars: %+v", chars)
				}
				if res.ScannedDocument.ScanID != "scan-id" || res.ScannedDocument.ActualCredits != 1 || res.ScannedDocument.TotalWords != 73 {
					t.Errorf("scannedDocument: %+v", res.ScannedDocument)
				}
				stats := res.Explain.Patterns.Statistics
				if len(stats.AICount) != 3 || stats.Proportion[2] != 102.54607 || !reflect.DeepEqual(stats.Source, []int{1, 1, 1}) {
					t.Errorf("explain statistics: %+v", stats)
				}
				if words := res.Explain.Patterns.Text.Words; !reflect.DeepEqual(words.Lengths, []int{4, 4, 5}) {
					t.Errorf("explain words: %+v", words)
				}
			},
		},
		{
			name:     "DetectAIImage",
			method:   http.MethodPost,
			path:     "/v1/ai-image-detector/scan-1/check",
			auth:     true,
			body:     `{"base64":"iVBORw0KGgo=","filename":"image.png","model":"ai-image-1-ultra","sandbox":true}`,
			status:   http.StatusOK,
			response: fixture(t, "ai_image_response.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.DetectAIImage(ctx, tok, "scan-1", &AIImageDetectionRequest{Base64: "iVBORw0KGgo=", Filename: "image.png", Model: AIImageModel1Ultra, Sandbox: true})
			},
			check: func(t *testing.T, got any) {
				res := got.(*AIImageDetectionResponse)
				if res.Model != "ai-image-1-ultra" || !res.IsAIDetected || res.Summary.AI != 0.75 || res.Summary.Human != 0.25 {
					t.Errorf("got %+v %+v", res, res.Summary)
				}
				if !reflect.DeepEqual(res.Result.Starts, []int{0, 393216}) || !reflect.DeepEqual(res.Result.Lengths, []int{196608, 393216}) {
					t.Errorf("result: %+v", res.Result)
				}
				if res.ImageInfo.Shape.Height != 1024 || res.ImageInfo.Shape.Width != 768 || res.ImageInfo.Metadata.IssuedBy != "OpenAI" {
					t.Errorf("imageInfo: %+v %+v", res.ImageInfo.Shape, res.ImageInfo.Metadata)
				}
				if res.ScannedDocument.ScanID != "my-scan-123" || res.ScannedDocument.ExpectedCredits != 1 {
					t.Errorf("scannedDocument: %+v", res.ScannedDocument)
				}
			},
		},
		{
			name:     "CheckGrammar",
			method:   http.MethodPost,
			path:     "/v1/writing-feedback/scan-1/check",
			auth:     true,
			body:     `{"text":"Its a test.","sandbox":true,"language":"en","score":{"grammarScoreWeight":0.2,"mechanicsScoreWeight":0}}`,
			status:   http.StatusOK,
			response: fixture(t, "grammar_response.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.CheckGrammar(ctx, tok, "scan-1", &GrammarCheckRequest{
					Text:     "Its a test.",
					Sandbox:  true,
					Language: "en",
					Score:    &GrammarScoreWeights{GrammarScoreWeight: Ptr(0.2), MechanicsScoreWeight: Ptr(0.0)},
				})
			},
			check: func(t *testing.T, got any) {
				res := got.(*GrammarCheckResponse)
				corr := res.Score.Corrections
				if corr.OverallScore != 79 || corr.GrammarCorrectionsCount != 2 || corr.MechanicsCorrectionsScore != 38 || corr.WordChoiceScoreWeight != 1 {
					t.Errorf("score corrections: %+v", corr)
				}
				if r := res.Score.Readability; r.Score != 59 || r.ReadabilityLevel != 5 || r.ReadabilityLevelText != "10th to 12th Grader" {
					t.Errorf("readability: %+v", r)
				}
				if s := res.Score.Statistics; s.SentenceCount != 5 || s.AverageSentenceLength != 12.2 || s.SpeakingTimeSeconds != 28.2 {
					t.Errorf("statistics: %+v", s)
				}
				chars := res.Corrections.Text.Chars
				if !reflect.DeepEqual(chars.Types, []int{5, 18, 3}) || !reflect.DeepEqual(chars.OperationTexts, []string{"an ", "plagiarism ", "businesses "}) {
					t.Errorf("corrections: %+v", chars)
				}
				if res.ScannedDocument.TotalWords != 61 {
					t.Errorf("scannedDocument: %+v", res.ScannedDocument)
				}
			},
		},
		{
			name:     "GrammarCorrectionTypes",
			method:   http.MethodGet,
			path:     "/v1/writing-feedback/correction-types/en",
			status:   http.StatusOK,
			response: fixture(t, "correction_types.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.GrammarCorrectionTypes(ctx, "en")
			},
			check: func(t *testing.T, got any) {
				want := &CorrectionTypes{CorrectionTypes: []CorrectionType{{
					ID:          1,
					Category:    2,
					Title:       "General",
					Message:     "A general correction detected",
					Description: "A general correction detected.",
				}}}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("got %+v, want %+v", got, want)
				}
			},
		},
		{
			name:   "ModerateText",
			method: http.MethodPost,
			path:   "/v1/text-moderation/scan-1/check",
			auth:   true,
			body: `{"text":"This is some text to scan.","sandbox":true,"language":"en",` +
				`"labels":[{"id":"toxic-v1"},{"id":"profanity-v1"}]}`,
			status:   http.StatusOK,
			response: fixture(t, "moderation_response.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.ModerateText(ctx, tok, "scan-1", &TextModerationRequest{
					Text:     "This is some text to scan.",
					Sandbox:  true,
					Language: TextModerationLanguageEnglish,
					Labels:   []TextModerationLabel{{ID: TextModerationLabelToxicV1}, {ID: TextModerationLabelProfanityV1}},
				})
			},
			check: func(t *testing.T, got any) {
				res := got.(*TextModerationResponse)
				chars := res.Moderations.Text.Chars
				if res.ModelVersion != "v1" || !reflect.DeepEqual(chars.Labels, []int{1, 1, 1, 0, 2, 2}) || chars.Lengths[5] != 24 {
					t.Errorf("moderations: %q %+v", res.ModelVersion, chars)
				}
				if len(res.Legend) != 3 || res.Legend[2] != (TextModerationLegend{Index: 2, ID: "hate-speech-v1"}) {
					t.Errorf("legend: %+v", res.Legend)
				}
				if res.ScannedDocument.TotalWords != 86 {
					t.Errorf("scannedDocument: %+v", res.ScannedDocument)
				}
			},
		},
	}
}

func TestEndpointRequestsAndResponses(t *testing.T) {
	for _, tc := range endpointCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, tc.status, tc.response)
			got, err := tc.call(context.Background(), env.client(t))
			if err != nil {
				t.Fatalf("call failed: %v", err)
			}
			req := env.next(t)

			wantServer := tc.server
			if wantServer == "" {
				wantServer = "api"
			}
			if req.server != wantServer || req.method != tc.method || req.path != tc.path {
				t.Errorf("request = %s server %s %s, want %s server %s %s", req.server, req.method, req.path, wantServer, tc.method, tc.path)
			}
			if tc.query != nil && !reflect.DeepEqual(req.query, tc.query) {
				t.Errorf("query = %v, want %v", req.query, tc.query)
			}
			wantAuth := ""
			if tc.auth {
				wantAuth = "Bearer " + testAccessToken
			}
			if got := req.header.Get("Authorization"); got != wantAuth {
				t.Errorf("Authorization = %q, want %q", got, wantAuth)
			}
			if got, want := req.header.Get("User-Agent"), "copyleaks-go-sdk/"+Version; got != want {
				t.Errorf("User-Agent = %q, want %q", got, want)
			}
			if values := req.header.Values("X-Tenant"); len(values) != 0 {
				t.Errorf("x-tenant = %q, want no header", values)
			}
			if tc.body == "" {
				if len(req.body) != 0 || req.header.Get("Content-Type") != "" {
					t.Errorf("unexpected body %q with Content-Type %q", req.body, req.header.Get("Content-Type"))
				}
			} else {
				if got := req.header.Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", got)
				}
				assertJSONEqual(t, req.body, tc.body)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestTenantHeaderOnEveryEndpoint(t *testing.T) {
	scenarios := []struct {
		name       string
		options    []Option
		callTenant *string
		want       string
	}{
		{"unset", nil, nil, ""},
		{"client tenant", []Option{WithTenant("t1")}, nil, "t1"},
		{"per-call tenant overrides client tenant", []Option{WithTenant("t1")}, Ptr("t2"), "t2"},
		{"per-call tenant without client tenant", nil, Ptr("t3"), "t3"},
		{"empty per-call tenant omits header", []Option{WithTenant("t1")}, Ptr(""), ""},
	}
	for _, tc := range endpointCases(t) {
		for _, sc := range scenarios {
			t.Run(tc.name+"/"+sc.name, func(t *testing.T) {
				env := newTestEnv(t, tc.status, tc.response)
				ctx := context.Background()
				if sc.callTenant != nil {
					ctx = WithCallTenant(ctx, *sc.callTenant)
				}
				if _, err := tc.call(ctx, env.client(t, sc.options...)); err != nil {
					t.Fatalf("call failed: %v", err)
				}
				values := env.next(t).header.Values("X-Tenant")
				switch {
				case sc.want == "" && len(values) != 0:
					t.Errorf("x-tenant = %q, want no header", values)
				case sc.want != "" && (len(values) != 1 || values[0] != sc.want):
					t.Errorf("x-tenant = %q, want %q", values, sc.want)
				}
			})
		}
	}
}

func TestTenantDoesNotLeakBetweenClients(t *testing.T) {
	env := newTestEnv(t, http.StatusOK, `["en"]`)
	withTenant := env.client(t, WithTenant("t1"))
	without := env.client(t)
	ctx := context.Background()
	if _, err := withTenant.OCRSupportedLanguages(ctx); err != nil {
		t.Fatal(err)
	}
	if got := env.next(t).header.Get("x-tenant"); got != "t1" {
		t.Errorf("x-tenant = %q, want t1", got)
	}
	if _, err := without.OCRSupportedLanguages(ctx); err != nil {
		t.Fatal(err)
	}
	if got := env.next(t).header.Values("x-tenant"); len(got) != 0 {
		t.Errorf("x-tenant = %q, want no header", got)
	}
}

func TestAITextDetectionOptionalFields(t *testing.T) {
	tests := []struct {
		name string
		req  AITextDetectionRequest
		body string
	}{
		{"neither", AITextDetectionRequest{Text: lionsText}, `{"text":"` + lionsText + `"}`},
		{"explain", AITextDetectionRequest{Text: lionsText, Explain: true}, `{"text":"` + lionsText + `","explain":true}`},
		{"sensitivity", AITextDetectionRequest{Text: lionsText, Sensitivity: Ptr(1)}, `{"text":"` + lionsText + `","sensitivity":1}`},
		{"both", AITextDetectionRequest{Text: lionsText, Explain: true, Sensitivity: Ptr(3)}, `{"text":"` + lionsText + `","explain":true,"sensitivity":3}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, "{}")
			if _, err := env.client(t).DetectAIText(context.Background(), testToken(), "scan-1", &tt.req); err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, env.next(t).body, tt.body)
		})
	}
}

func TestSubmitAISourceMatchDisplayLanguageReportVersion(t *testing.T) {
	webhooks := SubmitWebhooks{Status: "https://example.com/webhook/{STATUS}"}
	const webhooksJSON = `"webhooks":{"status":"https://example.com/webhook/{STATUS}"}`
	tests := []struct {
		name  string
		props ScanProperties
		body  string
	}{
		{"unset", ScanProperties{Webhooks: webhooks},
			`{` + webhooksJSON + `}`},
		{"unset with pdf", ScanProperties{Webhooks: webhooks, PDF: &PDF{Create: true}},
			`{` + webhooksJSON + `,"pdf":{"create":true}}`},
		{"aiSourceMatch enabled", ScanProperties{Webhooks: webhooks, AISourceMatch: &AISourceMatch{Enable: true}},
			`{` + webhooksJSON + `,"aiSourceMatch":{"enable":true}}`},
		{"aiSourceMatch disabled", ScanProperties{Webhooks: webhooks, AISourceMatch: &AISourceMatch{}},
			`{` + webhooksJSON + `,"aiSourceMatch":{"enable":false}}`},
		{"displayLanguage", ScanProperties{Webhooks: webhooks, DisplayLanguage: "es"},
			`{` + webhooksJSON + `,"displayLanguage":"es"}`},
		{"reportVersion", ScanProperties{Webhooks: webhooks, PDF: &PDF{Create: true, ReportVersion: PDFReportVersionLatest}},
			`{` + webhooksJSON + `,"pdf":{"create":true,"reportVersion":"latest"}}`},
		{"reportVersion and version", ScanProperties{Webhooks: webhooks, PDF: &PDF{Create: true, Version: PDFVersion1, ReportVersion: PDFReportVersionV3}},
			`{` + webhooksJSON + `,"pdf":{"create":true,"version":1,"reportVersion":"v3"}}`},
		{"all three", ScanProperties{
			Webhooks:        webhooks,
			PDF:             &PDF{Create: true, ReportVersion: PDFReportVersionV2},
			DisplayLanguage: "de",
			AISourceMatch:   &AISourceMatch{Enable: true},
		}, `{` + webhooksJSON + `,"pdf":{"create":true,"reportVersion":"v2"},"displayLanguage":"de","aiSourceMatch":{"enable":true}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusCreated, "")
			req := &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: tt.props}
			if err := env.client(t).SubmitFile(context.Background(), testToken(), "scan-1", req); err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, env.next(t).body, `{"base64":"aGk=","filename":"a.txt","properties":`+tt.body+`}`)
		})
	}
}

func TestDeleteHeaders(t *testing.T) {
	env := newTestEnv(t, http.StatusAccepted, "{}")
	req := &DeleteRequest{
		Scans:             []DeleteScan{{ID: "scan-1"}},
		CompletionWebhook: "https://example.com/deleted",
		Headers:           [][]string{{"x-key", "value-1"}, {"x-other", "value-2"}},
	}
	if err := env.client(t).Delete(context.Background(), testToken(), req); err != nil {
		t.Fatal(err)
	}
	want := `{"scans":[{"id":"scan-1"}],"completionWebhook":"https://example.com/deleted","headers":[["x-key","value-1"],["x-other","value-2"]]}`
	assertJSONEqual(t, env.next(t).body, want)
}
