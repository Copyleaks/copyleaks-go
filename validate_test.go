package copyleaks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// validSubmitProps are scan properties that pass the client-side checks.
var validSubmitProps = ScanProperties{Webhooks: SubmitWebhooks{Status: "https://example.com/webhook/{STATUS}"}, Sandbox: true}

func TestSubmitScanIDValidation(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"abc", true},
		{"scan-1!@$&=_()';:.,~", true},
		{strings.Repeat("a", 36), true},
		{"ab", false},
		{strings.Repeat("a", 37), false},
		{"Scan-1", false},
		{"scan 1", false},
		{"scan/1", false},
		{"scan#1", false},
		{"my-google-scan", false},
		{"gogle1", false},
		{"g00gle1", false},
		{"goo0ogle", false},
		{"ggle-1", true},
	}
	submits := map[string]func(c *Client, id string) error{
		"SubmitFile": func(c *Client, id string) error {
			return c.SubmitFile(context.Background(), testToken(), id, &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: validSubmitProps})
		},
		"SubmitOCR": func(c *Client, id string) error {
			return c.SubmitOCR(context.Background(), testToken(), id, &OCRSubmission{Base64: "aGk=", Filename: "a.png", LangCode: "en", Properties: validSubmitProps})
		},
		"SubmitURL": func(c *Client, id string) error {
			return c.SubmitURL(context.Background(), testToken(), id, &URLSubmission{URL: "https://example.com", Properties: validSubmitProps})
		},
	}
	for name, submit := range submits {
		for _, tt := range tests {
			t.Run(name+"/"+tt.id, func(t *testing.T) {
				env := newTestEnv(t, http.StatusCreated, "")
				err := submit(env.client(t), tt.id)
				if tt.valid {
					if err != nil {
						t.Fatalf("err = %v, want nil", err)
					}
					env.next(t)
					return
				}
				if !errors.Is(err, ErrInvalidArgument) {
					t.Fatalf("err = %v, want ErrInvalidArgument", err)
				}
				env.none(t)
			})
		}
	}
}

func TestOtherScanIDsOnlyCheckLength(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		id    string
		valid bool
	}{
		{"my-google-scan", true},
		{"Scan_ID-With-Upper", true},
		{"ab", false},
		{strings.Repeat("a", 37), false},
	} {
		t.Run(tt.id, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, "{}")
			_, err := env.client(t).DetectAIText(ctx, testToken(), tt.id, &AITextDetectionRequest{Text: lionsText})
			if tt.valid != (err == nil) {
				t.Fatalf("err = %v, want valid = %v", err, tt.valid)
			}
			if !tt.valid && !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("err = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestAITextLength(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		valid bool
	}{
		{"254 characters", strings.Repeat("a", 254), false},
		{"255 characters", strings.Repeat("a", 255), true},
		{"100000 characters", strings.Repeat("a", 100000), true},
		{"100001 characters", strings.Repeat("a", 100001), false},
		{"128 emoji are 256 UTF-16 units", strings.Repeat("\U0001F600", 128), true},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, "{}")
			_, err := env.client(t).DetectAIText(context.Background(), testToken(), "scan-1", &AITextDetectionRequest{Text: tt.text})
			if tt.valid {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
			env.none(t)
		})
	}
}

func TestAITextSensitivity(t *testing.T) {
	for sensitivity, valid := range map[int]bool{-1: false, 0: false, 1: true, 2: true, 3: true, 4: false} {
		env := newTestEnv(t, http.StatusOK, "{}")
		req := &AITextDetectionRequest{Text: lionsText, Sensitivity: Ptr(sensitivity)}
		_, err := env.client(t).DetectAIText(context.Background(), testToken(), "scan-1", req)
		if valid {
			if err != nil {
				t.Errorf("sensitivity %d: err = %v, want nil", sensitivity, err)
			}
			continue
		}
		if !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("sensitivity %d: err = %v, want ErrInvalidArgument", sensitivity, err)
		}
		env.none(t)
	}
}

func TestTextModerationTextLength(t *testing.T) {
	labels := []TextModerationLabel{{ID: TextModerationLabelToxicV1}}
	cases := map[string]bool{
		"":                                  false,
		"a":                                 false,
		"ab":                                true,
		"\U0001F600":                        true,
		strings.Repeat("a", 25000):          true,
		strings.Repeat("a", 25001):          false,
		strings.Repeat("\U0001F600", 12501): false,
	}
	for text, valid := range cases {
		env := newTestEnv(t, http.StatusOK, "{}")
		_, err := env.client(t).ModerateText(context.Background(), testToken(), "scan-1", &TextModerationRequest{Text: text, Labels: labels})
		if valid {
			if err != nil {
				t.Errorf("text of %d bytes: err = %v, want nil", len(text), err)
			}
			continue
		}
		if !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("text of %d bytes: err = %v, want ErrInvalidArgument", len(text), err)
		}
		env.none(t)
	}
}

func TestAIImageFilename(t *testing.T) {
	tests := []struct {
		filename string
		valid    bool
	}{
		{"a.png", true},
		{"IMAGE.JPEG", true},
		{"photo.heif", true},
		{"photo.avif", true},
		{".png", false},
		{"image.gif", false},
		{"image", false},
		{strings.Repeat("a", 250) + ".png", true},
		{strings.Repeat("a", 251) + ".png", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%.20s", tt.filename), func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, "{}")
			req := &AIImageDetectionRequest{Base64: "iVBORw0KGgo=", Filename: tt.filename, Model: AIImageModel1Ultra}
			_, err := env.client(t).DetectAIImage(context.Background(), testToken(), "scan-1", req)
			if tt.valid != (err == nil) {
				t.Fatalf("err = %v, want valid = %v", err, tt.valid)
			}
			if !tt.valid {
				env.none(t)
			}
		})
	}
}

func TestTextModerationLabels(t *testing.T) {
	labels := func(n int) []TextModerationLabel {
		out := make([]TextModerationLabel, n)
		for i := range out {
			out[i] = TextModerationLabel{ID: fmt.Sprintf("label-%d", i)}
		}
		return out
	}
	tests := []struct {
		name   string
		labels []TextModerationLabel
		valid  bool
	}{
		{"none", nil, false},
		{"one", labels(1), true},
		{"sixteen", labels(16), true},
		{"seventeen", labels(17), false},
		{"empty id", []TextModerationLabel{{ID: ""}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, "{}")
			req := &TextModerationRequest{Text: "Some text.", Labels: tt.labels}
			_, err := env.client(t).ModerateText(context.Background(), testToken(), "scan-1", req)
			if tt.valid != (err == nil) {
				t.Fatalf("err = %v, want valid = %v", err, tt.valid)
			}
			if !tt.valid && !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("err = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestRequiredArguments(t *testing.T) {
	ctx := context.Background()
	tok := testToken()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	noWebhook := ScanProperties{Sandbox: true}
	longPayload := validSubmitProps
	longPayload.DeveloperPayload = strings.Repeat("p", 513)
	badSensitivity := validSubmitProps
	badSensitivity.SensitivityLevel = 6
	longIDPattern := validSubmitProps
	longIDPattern.Scanning = &Scanning{Exclude: &ScanningExclude{IDPattern: strings.Repeat("a", 37)}}
	badDisplayLanguage := validSubmitProps
	badDisplayLanguage.DisplayLanguage = "EN"
	badReportVersion := validSubmitProps
	badReportVersion.PDF = &PDF{Create: true, ReportVersion: "V3"}
	tests := map[string]func(c *Client) error{
		"Login without email": func(c *Client) error {
			_, err := c.Login(ctx, "", "key")
			return err
		},
		"Login without key": func(c *Client) error {
			_, err := c.Login(ctx, "user@example.com", "")
			return err
		},
		"SubmitFile nil request": func(c *Client) error {
			return c.SubmitFile(ctx, tok, "scan-1", nil)
		},
		"SubmitFile without filename": func(c *Client) error {
			return c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Base64: "aGk=", Properties: validSubmitProps})
		},
		"SubmitFile without status webhook": func(c *Client) error {
			return c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: noWebhook})
		},
		"SubmitFile with a 513-character developer payload": func(c *Client) error {
			return c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: longPayload})
		},
		"SubmitFile with sensitivity level 6": func(c *Client) error {
			return c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: badSensitivity})
		},
		"SubmitURL with a 37-character idPattern": func(c *Client) error {
			return c.SubmitURL(ctx, tok, "scan-1", &URLSubmission{URL: "https://example.com", Properties: longIDPattern})
		},
		"SubmitFile with display language EN": func(c *Client) error {
			return c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: badDisplayLanguage})
		},
		"SubmitOCR with report version V3": func(c *Client) error {
			req := &OCRSubmission{Base64: "aGk=", Filename: "a.png", LangCode: "en", Properties: badReportVersion}
			return c.SubmitOCR(ctx, tok, "scan-1", req)
		},
		"SubmitOCR without language": func(c *Client) error {
			return c.SubmitOCR(ctx, tok, "scan-1", &OCRSubmission{Base64: "aGk=", Filename: "a.png", Properties: validSubmitProps})
		},
		"SubmitURL without URL": func(c *Client) error {
			return c.SubmitURL(ctx, tok, "scan-1", &URLSubmission{Properties: validSubmitProps})
		},
		"Export without completion webhook": func(c *Client) error {
			return c.Export(ctx, tok, "scan-1", "export-1", &ExportRequest{})
		},
		"Export with 13 retries": func(c *Client) error {
			return c.Export(ctx, tok, "scan-1", "export-1", &ExportRequest{CompletionWebhook: "https://example.com", MaxRetries: 13})
		},
		"Export result without verb": func(c *Client) error {
			req := &ExportRequest{CompletionWebhook: "https://example.com", Results: []ExportResult{{ID: "r1", Endpoint: "https://example.com"}}}
			return c.Export(ctx, tok, "scan-1", "export-1", req)
		},
		"Export PDF report without endpoint": func(c *Client) error {
			req := &ExportRequest{CompletionWebhook: "https://example.com", PDFReport: &ExportTarget{Verb: "POST"}}
			return c.Export(ctx, tok, "scan-1", "export-1", req)
		},
		"Export with a short export ID": func(c *Client) error {
			return c.Export(ctx, tok, "scan-1", "e1", &ExportRequest{CompletionWebhook: "https://example.com"})
		},
		"Start without trigger": func(c *Client) error {
			_, err := c.Start(ctx, tok, &StartRequest{})
			return err
		},
		"Start with 101 scans": func(c *Client) error {
			_, err := c.Start(ctx, tok, &StartRequest{Trigger: make([]string, 101)})
			return err
		},
		"Delete without scans": func(c *Client) error {
			return c.Delete(ctx, tok, &DeleteRequest{})
		},
		"Delete scan without ID": func(c *Client) error {
			return c.Delete(ctx, tok, &DeleteRequest{Scans: []DeleteScan{{}}})
		},
		"ResendWebhook with a short scan ID": func(c *Client) error {
			return c.ResendWebhook(ctx, tok, "s1")
		},
		"UsagesHistoryCSV without dates": func(c *Client) error {
			_, err := c.UsagesHistoryCSV(ctx, tok, time.Time{}, day)
			return err
		},
		"DetectAIText nil request": func(c *Client) error {
			_, err := c.DetectAIText(ctx, tok, "scan-1", nil)
			return err
		},
		"DetectAIImage without model": func(c *Client) error {
			_, err := c.DetectAIImage(ctx, tok, "scan-1", &AIImageDetectionRequest{Base64: "aGk=", Filename: "a.png"})
			return err
		},
		"CheckGrammar without text": func(c *Client) error {
			_, err := c.CheckGrammar(ctx, tok, "scan-1", &GrammarCheckRequest{})
			return err
		},
		"GrammarCorrectionTypes without language": func(c *Client) error {
			_, err := c.GrammarCorrectionTypes(ctx, "")
			return err
		},
		"ModerateText without text": func(c *Client) error {
			_, err := c.ModerateText(ctx, tok, "scan-1", &TextModerationRequest{Labels: []TextModerationLabel{{ID: TextModerationLabelToxicV1}}})
			return err
		},
	}
	for name, call := range tests {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, "{}")
			if err := call(env.client(t)); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
			env.none(t)
		})
	}
}

// authenticatedCalls calls every method that needs a token, with token.
func authenticatedCalls(token *AuthToken) map[string]func(c *Client) error {
	ctx := context.Background()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return map[string]func(c *Client) error{
		"SubmitFile": func(c *Client) error {
			return c.SubmitFile(ctx, token, "scan-1", &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: validSubmitProps})
		},
		"SubmitOCR": func(c *Client) error {
			return c.SubmitOCR(ctx, token, "scan-1", &OCRSubmission{Base64: "aGk=", Filename: "a.png", LangCode: "en", Properties: validSubmitProps})
		},
		"SubmitURL": func(c *Client) error {
			return c.SubmitURL(ctx, token, "scan-1", &URLSubmission{URL: "https://example.com", Properties: validSubmitProps})
		},
		"Export": func(c *Client) error {
			return c.Export(ctx, token, "scan-1", "export-1", &ExportRequest{CompletionWebhook: "https://example.com"})
		},
		"Start": func(c *Client) error {
			_, err := c.Start(ctx, token, &StartRequest{Trigger: []string{"scan-1"}})
			return err
		},
		"Delete": func(c *Client) error {
			return c.Delete(ctx, token, &DeleteRequest{Scans: []DeleteScan{{ID: "scan-1"}}})
		},
		"ResendWebhook": func(c *Client) error {
			return c.ResendWebhook(ctx, token, "scan-1")
		},
		"CreditsBalance": func(c *Client) error {
			_, err := c.CreditsBalance(ctx, token)
			return err
		},
		"UsagesHistoryCSV": func(c *Client) error {
			_, err := c.UsagesHistoryCSV(ctx, token, day, day)
			return err
		},
		"DetectAIText": func(c *Client) error {
			_, err := c.DetectAIText(ctx, token, "scan-1", &AITextDetectionRequest{Text: lionsText})
			return err
		},
		"DetectAIImage": func(c *Client) error {
			_, err := c.DetectAIImage(ctx, token, "scan-1", &AIImageDetectionRequest{Base64: "aGk=", Filename: "a.png", Model: AIImageModel1Ultra})
			return err
		},
		"CheckGrammar": func(c *Client) error {
			_, err := c.CheckGrammar(ctx, token, "scan-1", &GrammarCheckRequest{Text: "Some text."})
			return err
		},
		"ModerateText": func(c *Client) error {
			_, err := c.ModerateText(ctx, token, "scan-1", &TextModerationRequest{Text: "Some text.", Labels: []TextModerationLabel{{ID: TextModerationLabelToxicV1}}})
			return err
		},
	}
}

func TestAuthenticatedCallsCheckTheToken(t *testing.T) {
	tests := []struct {
		name   string
		token  *AuthToken
		target error
	}{
		{"nil token", nil, ErrInvalidArgument},
		{"empty access token", &AuthToken{Expires: time.Now().Add(time.Hour)}, ErrInvalidArgument},
		{"expired token", &AuthToken{AccessToken: "x", Expires: time.Now().Add(-time.Hour)}, ErrTokenExpired},
		{"token expiring in four minutes", &AuthToken{AccessToken: "x", Expires: time.Now().Add(4 * time.Minute)}, ErrTokenExpired},
	}
	for _, tt := range tests {
		for name, call := range authenticatedCalls(tt.token) {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				env := newTestEnv(t, http.StatusOK, "{}")
				if err := call(env.client(t)); !errors.Is(err, tt.target) {
					t.Fatalf("err = %v, want %v", err, tt.target)
				}
				env.none(t)
			})
		}
	}
}

func TestVerifyAuthToken(t *testing.T) {
	if err := VerifyAuthToken(&AuthToken{AccessToken: "x", Expires: time.Now().Add(6 * time.Minute)}); err != nil {
		t.Errorf("token expiring in six minutes: err = %v, want nil", err)
	}
	if err := VerifyAuthToken(&AuthToken{AccessToken: "x"}); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("token without expiry: err = %v, want ErrTokenExpired", err)
	}
	if err := VerifyAuthToken(nil); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("nil token: err = %v, want ErrInvalidArgument", err)
	}
}

func TestAuthTokenJSON(t *testing.T) {
	var tok AuthToken
	login := `{"access_token":"abc",".issued":"2018-11-24T16:15:38.2431255+02:00",".expires":"2018-11-26T16:15:38.2431255+02:00"}`
	if err := json.Unmarshal([]byte(login), &tok); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2018, 11, 26, 14, 15, 38, 243125500, time.UTC); tok.AccessToken != "abc" || !tok.Expires.Equal(want) {
		t.Errorf("token = %q expiring %v, want %v", tok.AccessToken, tok.Expires, want)
	}

	if err := json.Unmarshal([]byte(`{"access_token":"abc",".expires":"2030-01-02T03:04:05.5"}`), &tok); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2030, 1, 2, 3, 4, 5, 500000000, time.UTC); !tok.Expires.Equal(want) || !tok.Issued.IsZero() {
		t.Errorf("time without offset: expires %v, issued %v", tok.Expires, tok.Issued)
	}

	stored, err := json.Marshal(tok)
	if err != nil {
		t.Fatal(err)
	}
	var restored AuthToken
	if err := json.Unmarshal(stored, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.AccessToken != tok.AccessToken || !restored.Expires.Equal(tok.Expires) {
		t.Errorf("round trip: %s", stored)
	}

	if err := json.Unmarshal([]byte(`{"access_token":"abc",".expires":"next tuesday"}`), &tok); err == nil {
		t.Error("an unparsable expiry must fail")
	}
}

func TestAuthTokenPrintingRedactsTheAccessToken(t *testing.T) {
	tok := testToken()
	verbs := []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d", "%t", "%e", "%f", "%g", "%c", "%U", "%o", "%b", "%10.3v", "%-40s"}
	values := []any{tok, *tok, struct{ Token *AuthToken }{Token: tok}, []AuthToken{*tok}}
	for _, format := range verbs {
		for _, v := range values {
			if s := fmt.Sprintf(format, v); strings.Contains(s, testAccessToken) || !strings.Contains(s, "[REDACTED]") {
				t.Errorf("Sprintf(%q, %T) = %q", format, v, s)
			}
		}
	}
	for _, s := range []string{fmt.Sprint(tok), fmt.Sprintln(*tok), tok.String()} {
		if strings.Contains(s, testAccessToken) || !strings.Contains(s, "[REDACTED]") {
			t.Errorf("printed token = %q", s)
		}
	}
}

func TestExportIDValidation(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"export-1", true},
		{"a^b+c%d{e}<f>|g~h", true},
		{"it's;ok:\"`\\", true},
		{"ab", false},
		{strings.Repeat("a", 37), false},
		{"Export-1", false},
		{"export 1", false},
		{"export#1", false},
		{"export[1]", false},
		{"my-google-export", false},
		{"g00gle-1", false},
	}
	req := &ExportRequest{CompletionWebhook: "https://example.com/done"}
	for _, tt := range tests {
		for _, which := range []string{"scan ID", "export ID"} {
			t.Run(which+"/"+tt.id, func(t *testing.T) {
				env := newTestEnv(t, http.StatusNoContent, "")
				scanID, exportID := "scan-1", "export-1"
				if which == "scan ID" {
					scanID = tt.id
				} else {
					exportID = tt.id
				}
				err := env.client(t).Export(context.Background(), testToken(), scanID, exportID, req)
				if tt.valid {
					if err != nil {
						t.Fatalf("err = %v, want nil", err)
					}
					env.next(t)
					return
				}
				if !errors.Is(err, ErrInvalidArgument) || !strings.Contains(err.Error(), which) {
					t.Fatalf("err = %v, want ErrInvalidArgument for the %s", err, which)
				}
				env.none(t)
			})
		}
	}
}

func TestScanningExcludeIDPatternLength(t *testing.T) {
	for n, valid := range map[int]bool{1: true, 36: true, 37: false} {
		props := validSubmitProps
		props.Scanning = &Scanning{Exclude: &ScanningExclude{IDPattern: strings.Repeat("a", n)}}
		err := props.validate()
		if valid != (err == nil) {
			t.Errorf("%d characters: err = %v, want valid = %v", n, err, valid)
		}
		if !valid && !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%d characters: err = %v, want ErrInvalidArgument", n, err)
		}
	}
}

func TestDisplayLanguageValidation(t *testing.T) {
	// The server checks ^(es|pt|fr|de|it|en)$, case-sensitive, on the whole value.
	cases := map[string]bool{
		"": true, "en": true, "es": true, "pt": true, "fr": true, "de": true, "it": true,
		"EN": false, "En": false, "ja": false, "en-US": false, " en": false, "en\n": false,
	}
	for lang, valid := range cases {
		props := validSubmitProps
		props.DisplayLanguage = lang
		err := props.validate()
		if valid != (err == nil) {
			t.Errorf("%q: err = %v, want valid = %v", lang, err, valid)
		}
		if !valid && !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%q: err = %v, want ErrInvalidArgument", lang, err)
		}
	}
}

func TestPDFReportVersionValidation(t *testing.T) {
	// The server allows exactly v1, v2, v3 and latest, case-sensitive.
	cases := map[string]bool{
		"": true, "v1": true, "v2": true, "v3": true, "latest": true,
		"V1": false, "Latest": false, "LATEST": false, "v0": false, "v4": false, "3": false, "v3 ": false,
	}
	for version, valid := range cases {
		props := validSubmitProps
		props.PDF = &PDF{Create: true, ReportVersion: version}
		err := props.validate()
		if valid != (err == nil) {
			t.Errorf("%q: err = %v, want valid = %v", version, err, valid)
		}
		if !valid && !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%q: err = %v, want ErrInvalidArgument", version, err)
		}
	}
}

func TestScanPropertiesJSONNames(t *testing.T) {
	props := ScanProperties{
		Webhooks: SubmitWebhooks{
			Status:           "https://example.com/{STATUS}",
			StatusHeaders:    [][]string{{"k", "v"}},
			NewResult:        "https://example.com/new",
			NewResultHeaders: [][]string{{"k2", "v2"}},
		},
		Sandbox:          true,
		Action:           SubmitActionCheckCredits,
		IncludeHTML:      true,
		DeveloperPayload: "payload",
		Expiration:       100,
		Author:           &Author{ID: "author-1"},
		Filters: &Filters{
			IdenticalEnabled:      Ptr(false),
			MinorChangesEnabled:   Ptr(true),
			RelatedMeaningEnabled: Ptr(false),
			MinCopiedWords:        5,
			SafeSearch:            true,
			Domains:               []string{"example.com"},
			DomainsMode:           Ptr(DomainsModeInclude),
			AllowSameDomain:       true,
		},
		Scanning: &Scanning{
			Internet:       Ptr(false),
			Exclude:        &ScanningExclude{IDPattern: "abc*"},
			Repositories:   []ScanningRepository{{ID: "repo-1", IncludeMySubmissions: true, IncludeOthersSubmissions: true}},
			CopyleaksDB:    &CopyleaksDB{IncludeMySubmissions: true, IncludeOthersSubmissions: true},
			CrossLanguages: &CrossLanguages{Languages: []CrossLanguage{{Code: "fr"}}},
		},
		Indexing: &Indexing{Repositories: []IndexingRepository{{ID: "repo-1", MaskingPolicy: Ptr(MaskingPolicyNoMask)}}},
		Exclude: &Exclude{
			Quotes:              true,
			Citations:           true,
			References:          true,
			TableOfContents:     true,
			Titles:              true,
			HTMLTemplate:        true,
			DocumentTemplateIDs: []string{"scan-0"},
			Code:                &ExcludeCode{Comments: true},
		},
		PDF: &PDF{
			Create:        true,
			Title:         "Report",
			LargeLogo:     "aGk=",
			RTL:           true,
			Version:       PDFVersion2,
			ReportVersion: PDFReportVersionLatest,
			Colors: &PDFColors{
				MainStrip:      "#000001",
				Titles:         "#000002",
				Identical:      "#000003",
				MinorChanges:   "#000004",
				RelatedMeaning: "#000005",
			},
		},
		SensitivityLevel:    4,
		CheatDetection:      true,
		ScanMethodAlgorithm: ScanMethodMaximumResults,
		AIGeneratedText:     &AIGeneratedText{Detect: true},
		AISourceMatch:       &AISourceMatch{Enable: true},
		DisplayLanguage:     "fr",
		CustomMetadata:      []CustomMetadata{{Key: "k", Value: "v"}},
	}
	want := `{
		"webhooks": {"status": "https://example.com/{STATUS}", "statusHeaders": [["k", "v"]],
			"newResult": "https://example.com/new", "newResultHeaders": [["k2", "v2"]]},
		"sandbox": true, "action": 1, "includeHtml": true, "developerPayload": "payload", "expiration": 100,
		"author": {"id": "author-1"},
		"filters": {"identicalEnabled": false, "minorChangesEnabled": true, "relatedMeaningEnabled": false,
			"minCopiedWords": 5, "safeSearch": true, "domains": ["example.com"], "domainsMode": 0, "allowSameDomain": true},
		"scanning": {"internet": false, "exclude": {"idPattern": "abc*"},
			"repositories": [{"id": "repo-1", "includeMySubmissions": true, "includeOthersSubmissions": true}],
			"copyleaksDb": {"includeMySubmissions": true, "includeOthersSubmissions": true},
			"crossLanguages": {"languages": [{"code": "fr"}]}},
		"indexing": {"repositories": [{"id": "repo-1", "maskingPolicy": 0}]},
		"exclude": {"quotes": true, "citations": true, "references": true, "tableOfContents": true, "titles": true,
			"htmlTemplate": true, "documentTemplateIds": ["scan-0"], "code": {"comments": true}},
		"pdf": {"create": true, "title": "Report", "largeLogo": "aGk=", "rtl": true, "version": 2, "reportVersion": "latest",
			"colors": {"mainStrip": "#000001", "titles": "#000002", "identical": "#000003", "minorChanges": "#000004",
				"relatedMeaning": "#000005"}},
		"sensitivityLevel": 4, "cheatDetection": true, "scanMethodAlgorithm": 1,
		"displayLanguage": "fr",
		"aiGeneratedText": {"detect": true},
		"aiSourceMatch": {"enable": true},
		"customMetadata": [{"key": "k", "value": "v"}]
	}`
	got, err := json.Marshal(props)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, got, want)

	minimal, err := json.Marshal(ScanProperties{Webhooks: SubmitWebhooks{Status: "https://example.com/{STATUS}"}})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, minimal, `{"webhooks": {"status": "https://example.com/{STATUS}"}}`)
}
