package copyleaks

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// scriptedServer answers the requests that reach it with the steps of a script, in
// order, and repeats the last step. It records every request.
type scriptedServer struct {
	t     *testing.T
	srv   *httptest.Server
	mu    sync.Mutex
	steps []step
	got   []captured
}

// step is one answer of a scriptedServer: an HTTP status with headers and a body, or,
// when hangUp is set, a dropped connection after the whole request was read.
type step struct {
	status int
	header map[string]string
	body   string
	hangUp bool
}

func newScriptedServer(t *testing.T, steps ...step) *scriptedServer {
	t.Helper()
	s := &scriptedServer{t: t, steps: steps}
	s.srv = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *scriptedServer) serve(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	s.got = append(s.got, captured{method: r.Method, path: r.URL.Path, query: r.URL.Query(), header: r.Header.Clone(), body: b})
	st := s.steps[min(len(s.got), len(s.steps))-1]
	s.mu.Unlock()
	if st.hangUp {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			s.t.Errorf("hijack: %v", err)
			return
		}
		conn.Close()
		return
	}
	for k, v := range st.header {
		w.Header().Set(k, v)
	}
	w.WriteHeader(st.status)
	_, _ = io.WriteString(w, st.body)
}

// requests returns the requests that reached the server.
func (s *scriptedServer) requests() []captured {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]captured(nil), s.got...)
}

// retryClient returns a Client for s with policy, and the delays that the Client waited
// for, without waiting.
func (s *scriptedServer) retryClient(t *testing.T, policy RetryPolicy) (*Client, *[]time.Duration) {
	t.Helper()
	return retryClientFor(t, s.srv.URL, policy)
}

// retryClientFor returns a Client for the server at baseURL with policy, and the delays
// that the Client waited for, without waiting. It replaces sleep until the test ends.
func retryClientFor(t *testing.T, baseURL string, policy RetryPolicy) (*Client, *[]time.Duration) {
	t.Helper()
	c := mustClient(t, WithAPIBaseURL(baseURL), WithIdentityBaseURL(baseURL), WithRetry(policy))
	var slept []time.Duration
	old := sleep
	t.Cleanup(func() { sleep = old })
	sleep = func(ctx context.Context, d time.Duration) error {
		slept = append(slept, d)
		return ctx.Err()
	}
	return c, &slept
}

var (
	ok200       = step{status: http.StatusOK, body: "{}"}
	created201  = step{status: http.StatusCreated}
	rateLimited = step{status: http.StatusTooManyRequests, body: "slow down"}
	maintenance = step{status: http.StatusServiceUnavailable, body: "maintenance"}
	hangUp      = step{hangUp: true}
)

func TestRetryHonoursRetryAfterSeconds(t *testing.T) {
	s := newScriptedServer(t,
		step{status: http.StatusTooManyRequests, header: map[string]string{"Retry-After": "7"}},
		step{status: http.StatusServiceUnavailable, header: map[string]string{"Retry-After": "3"}},
		created201)
	c, slept := s.retryClient(t, RetryPolicy{MaxAttempts: 3})
	req := &FileSubmission{Content: []byte("hello world"), Filename: "a.txt", Properties: validSubmitProps}
	if err := c.SubmitFile(context.Background(), testToken(), "scan-1", req); err != nil {
		t.Fatalf("SubmitFile: %v", err)
	}
	got := s.requests()
	if len(got) != 3 {
		t.Fatalf("%d requests, want 3", len(got))
	}
	for i, r := range got {
		if r.method != http.MethodPut || r.path != "/v3/scans/submit/file/scan-1" || r.header.Get("Authorization") != "Bearer "+testAccessToken {
			t.Errorf("request %d = %s %s", i, r.method, r.path)
		}
		if string(r.body) != string(got[0].body) || len(r.body) == 0 {
			t.Errorf("request %d sent another body: %q", i, r.body)
		}
	}
	if want := []time.Duration{7 * time.Second, 3 * time.Second}; !equalDurations(*slept, want) {
		t.Errorf("waited %v, want %v", *slept, want)
	}
}

func TestRetryHonoursRetryAfterDate(t *testing.T) {
	when := time.Now().Add(20 * time.Second).UTC().Format(http.TimeFormat)
	s := newScriptedServer(t, step{status: http.StatusTooManyRequests, header: map[string]string{"Retry-After": when}}, ok200)
	c, slept := s.retryClient(t, RetryPolicy{MaxAttempts: 2})
	if _, err := c.CreditsBalance(context.Background(), testToken()); err != nil {
		t.Fatal(err)
	}
	if len(*slept) != 1 || (*slept)[0] < 15*time.Second || (*slept)[0] > 20*time.Second {
		t.Errorf("waited %v, want about 20s", *slept)
	}
}

func TestRetryBackoffWithJitter(t *testing.T) {
	s := newScriptedServer(t, maintenance, maintenance, maintenance, ok200)
	c, slept := s.retryClient(t, RetryPolicy{MaxAttempts: 4, BaseDelay: 100 * time.Millisecond, MaxDelay: 300 * time.Millisecond})
	if _, err := c.CreditsBalance(context.Background(), testToken()); err != nil {
		t.Fatal(err)
	}
	if len(s.requests()) != 4 || len(*slept) != 3 {
		t.Fatalf("%d requests and %d waits, want 4 and 3", len(s.requests()), len(*slept))
	}
	bounds := [][2]time.Duration{{50, 100}, {100, 200}, {150, 300}}
	for i, d := range *slept {
		lo, hi := bounds[i][0]*time.Millisecond, bounds[i][1]*time.Millisecond
		if d < lo || d > hi {
			t.Errorf("wait %d = %v, want %v to %v", i, d, lo, hi)
		}
	}
}

func TestRetryGivesUpAfterMaxAttempts(t *testing.T) {
	s := newScriptedServer(t, maintenance)
	c, slept := s.retryClient(t, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
	_, err := c.CreditsBalance(context.Background(), testToken())
	var apiErr *APIError
	if !errors.Is(err, ErrUnderMaintenance) || !errors.As(err, &apiErr) || string(apiErr.Body) != "maintenance" {
		t.Fatalf("err = %v, want the last *APIError", err)
	}
	if len(s.requests()) != 3 || len(*slept) != 2 {
		t.Errorf("%d requests and %d waits, want 3 and 2", len(s.requests()), len(*slept))
	}
}

func TestRetryAfterLongerThanMaxDelayIsNotWaitedFor(t *testing.T) {
	s := newScriptedServer(t, step{status: http.StatusTooManyRequests, header: map[string]string{"Retry-After": "120"}}, ok200)
	c, slept := s.retryClient(t, RetryPolicy{MaxAttempts: 5, MaxDelay: 30 * time.Second})
	_, err := c.CreditsBalance(context.Background(), testToken())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != 120*time.Second || !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want the 429 with RetryAfter 2m", err)
	}
	if len(s.requests()) != 1 || len(*slept) != 0 {
		t.Errorf("%d requests and %d waits, want 1 and 0", len(s.requests()), len(*slept))
	}
}

func TestRetryNotForOtherStatuses(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusConflict, http.StatusInternalServerError, http.StatusBadGateway} {
		s := newScriptedServer(t, step{status: status}, ok200)
		c, _ := s.retryClient(t, RetryPolicy{MaxAttempts: 3})
		if _, err := c.CreditsBalance(context.Background(), testToken()); err == nil {
			t.Errorf("HTTP %d: err = nil", status)
		}
		if n := len(s.requests()); n != 1 {
			t.Errorf("HTTP %d: %d requests, want 1", status, n)
		}
	}
}

// retryCase is one SDK call, for the tests of which calls are retried.
type retryCase struct {
	name string
	call func(ctx context.Context, c *Client) error
}

func safeCalls() []retryCase {
	tok := testToken()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ignore := func(_ any, err error) error { return err }
	return []retryCase{
		{"Login", func(ctx context.Context, c *Client) error { return ignore(c.Login(ctx, "a@example.com", "key")) }},
		{"SubmitFile", func(ctx context.Context, c *Client) error {
			return c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: validSubmitProps})
		}},
		{"SubmitOCR", func(ctx context.Context, c *Client) error {
			return c.SubmitOCR(ctx, tok, "scan-1", &OCRSubmission{Content: []byte{1, 2}, Filename: "a.png", LangCode: "en", Properties: validSubmitProps})
		}},
		{"SubmitURL", func(ctx context.Context, c *Client) error {
			return c.SubmitURL(ctx, tok, "scan-1", &URLSubmission{URL: "https://example.com", Properties: validSubmitProps})
		}},
		{"SubmitAIImageAsync", func(ctx context.Context, c *Client) error {
			return c.SubmitAIImageAsync(ctx, tok, "scan-1", &AIImageAsyncSubmission{URL: "https://example.com/a.png", FileName: "a.png", Model: AIImageModel1Ultra, Webhooks: ResultWebhook{URL: "https://example.com/hook"}})
		}},
		{"SubmitAIVideo", func(ctx context.Context, c *Client) error {
			return c.SubmitAIVideo(ctx, tok, "scan-1", &AIVideoSubmission{URL: "https://example.com/a.mp4", Filename: "a.mp4", Model: AIVideoModel1Pro, Webhooks: ResultWebhook{URL: "https://example.com/hook"}})
		}},
		{"Export", func(ctx context.Context, c *Client) error {
			return c.Export(ctx, tok, "scan-1", "export-1", &ExportRequest{CompletionWebhook: "https://example.com/done"})
		}},
		{"CreditsBalance", func(ctx context.Context, c *Client) error { return ignore(c.CreditsBalance(ctx, tok)) }},
		{"UsagesHistoryCSV", func(ctx context.Context, c *Client) error { return ignore(c.UsagesHistoryCSV(ctx, tok, day, day)) }},
		{"AITextCreditsBalance", func(ctx context.Context, c *Client) error { return ignore(c.AITextCreditsBalance(ctx, tok)) }},
		{"RepositoryInfo", func(ctx context.Context, c *Client) error { return ignore(c.RepositoryInfo(ctx, tok, "repo-1")) }},
		{"ReleaseNotes", func(ctx context.Context, c *Client) error { return ignore(c.ReleaseNotes(ctx)) }},
		{"SupportedFileTypes", func(ctx context.Context, c *Client) error { return ignore(c.SupportedFileTypes(ctx)) }},
		{"SupportedAITextFileTypes", func(ctx context.Context, c *Client) error { return ignore(c.SupportedAITextFileTypes(ctx)) }},
		{"AllowedCrossLanguages", func(ctx context.Context, c *Client) error { return ignore(c.AllowedCrossLanguages(ctx)) }},
		{"GrammarCorrectionTypes", func(ctx context.Context, c *Client) error { return ignore(c.GrammarCorrectionTypes(ctx, "en")) }},
	}
}

func unsafeCalls() []retryCase {
	tok := testToken()
	ignore := func(_ any, err error) error { return err }
	labels := []TextModerationLabel{{ID: TextModerationLabelToxicV1}}
	return []retryCase{
		{"DetectAIText", func(ctx context.Context, c *Client) error {
			return ignore(c.DetectAIText(ctx, tok, "scan-1", &AITextDetectionRequest{Text: lionsText}))
		}},
		{"DetectAIImage", func(ctx context.Context, c *Client) error {
			return ignore(c.DetectAIImage(ctx, tok, "scan-1", &AIImageDetectionRequest{Base64: "aGk=", Filename: "a.png", Model: AIImageModel1Ultra}))
		}},
		{"DetectAIImageUpload", func(ctx context.Context, c *Client) error {
			return ignore(c.DetectAIImageUpload(ctx, tok, "scan-1", &AIImageUploadRequest{Image: []byte{1}, Filename: "a.png", Model: AIImageModel1Ultra}))
		}},
		{"CheckImagePlagiarism", func(ctx context.Context, c *Client) error {
			return ignore(c.CheckImagePlagiarism(ctx, tok, "scan-1", &ImagePlagiarismRequest{Image: []byte{1}, Filename: "a.png"}))
		}},
		{"CheckGrammar", func(ctx context.Context, c *Client) error {
			return ignore(c.CheckGrammar(ctx, tok, "scan-1", &GrammarCheckRequest{Text: "Some text."}))
		}},
		{"ModerateText", func(ctx context.Context, c *Client) error {
			return ignore(c.ModerateText(ctx, tok, "scan-1", &TextModerationRequest{Text: "Some text.", Labels: labels}))
		}},
		{"Start", func(ctx context.Context, c *Client) error {
			return ignore(c.Start(ctx, tok, &StartRequest{Trigger: []string{"scan-1"}}))
		}},
		{"Delete", func(ctx context.Context, c *Client) error {
			return c.Delete(ctx, tok, &DeleteRequest{Scans: []DeleteScan{{ID: "scan-1"}}})
		}},
		{"ResendWebhook", func(ctx context.Context, c *Client) error { return c.ResendWebhook(ctx, tok, "scan-1") }},
	}
}

func TestRetrySafeCalls(t *testing.T) {
	for _, first := range []struct {
		name string
		step step
	}{{"429", rateLimited}, {"503", maintenance}, {"network error", hangUp}} {
		for _, tc := range safeCalls() {
			if tc.name == "Export" && first.step.status == http.StatusServiceUnavailable {
				continue // see TestRetryExportNotAfter503
			}
			t.Run(first.name+"/"+tc.name, func(t *testing.T) {
				s := newScriptedServer(t, first.step, ok200)
				c, slept := s.retryClient(t, RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond})
				_ = tc.call(context.Background(), c)
				got := s.requests()
				if len(got) != 2 || len(*slept) != 1 {
					t.Fatalf("%d requests and %d waits, want 2 and 1", len(got), len(*slept))
				}
				if got[0].method != got[1].method || got[0].path != got[1].path || string(got[0].body) != string(got[1].body) {
					t.Errorf("the retry differs: %s %s %q, then %s %s %q", got[0].method, got[0].path, got[0].body, got[1].method, got[1].path, got[1].body)
				}
			})
		}
	}
}

func TestRetryNeverRetriesUnsafeCalls(t *testing.T) {
	for _, first := range []struct {
		name string
		step step
	}{{"429", rateLimited}, {"503", maintenance}, {"network error", hangUp}} {
		for _, tc := range unsafeCalls() {
			t.Run(first.name+"/"+tc.name, func(t *testing.T) {
				s := newScriptedServer(t, first.step, ok200)
				c, slept := s.retryClient(t, RetryPolicy{MaxAttempts: 5, BaseDelay: time.Millisecond})
				if err := tc.call(context.Background(), c); err == nil {
					t.Fatal("err = nil, want the error of the first attempt")
				}
				if n := len(s.requests()); n != 1 || len(*slept) != 0 {
					t.Errorf("%d requests and %d waits, want 1 and 0", n, len(*slept))
				}
			})
		}
	}
}

func TestRetryExportNotAfter503(t *testing.T) {
	// After a 503 the export ID belongs to a failed export, so a retry would get a 409
	// that hides the 503.
	s := newScriptedServer(t, maintenance, step{status: http.StatusConflict, body: `{"title":"Conflict"}`})
	c, slept := s.retryClient(t, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
	err := c.Export(context.Background(), testToken(), "scan-1", "export-1", &ExportRequest{CompletionWebhook: "https://example.com/done"})
	if !errors.Is(err, ErrUnderMaintenance) {
		t.Fatalf("err = %v, want the 503", err)
	}
	if n := len(s.requests()); n != 1 || len(*slept) != 0 {
		t.Errorf("%d requests and %d waits, want 1 and 0", n, len(*slept))
	}
}

func TestRetryNotForCertificateErrors(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
	}))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	t.Cleanup(srv.Close)
	// The default client does not trust the test server's certificate.
	c, slept := retryClientFor(t, srv.URL, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
	_, err := c.CreditsBalance(context.Background(), testToken())
	var certErr *tls.CertificateVerificationError
	if !errors.As(err, &certErr) {
		t.Fatalf("err = %v, want a certificate error", err)
	}
	if requests.Load() != 0 || len(*slept) != 0 {
		t.Errorf("%d requests and %d waits, want 0 and 0", requests.Load(), len(*slept))
	}
}

func TestRetryOffByDefault(t *testing.T) {
	for _, first := range []step{rateLimited, maintenance, hangUp} {
		s := newScriptedServer(t, first, ok200)
		c := mustClient(t, WithAPIBaseURL(s.srv.URL))
		if _, err := c.CreditsBalance(context.Background(), testToken()); err == nil {
			t.Error("err = nil")
		}
		if n := len(s.requests()); n != 1 {
			t.Errorf("%d requests, want 1", n)
		}
	}
}

func TestRetryConflictAfterDroppedConnectionIsSuccess(t *testing.T) {
	// The first attempt was sent in full, then the connection dropped: the server may
	// have created the scan, so the 409 of the retry means it exists.
	s := newScriptedServer(t, hangUp, step{status: http.StatusConflict, body: `{"title":"Conflict"}`})
	c, _ := s.retryClient(t, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
	req := &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: validSubmitProps}
	if err := c.SubmitFile(context.Background(), testToken(), "scan-1", req); err != nil {
		t.Fatalf("SubmitFile: %v, want nil", err)
	}
	if n := len(s.requests()); n != 2 {
		t.Errorf("%d requests, want 2", n)
	}
}

func TestRetryConflictAfterRejectedAttemptIsAnError(t *testing.T) {
	// A 429 or 503 means that the server did not create the scan, so a 409 afterwards
	// is a real conflict with an older scan.
	for _, first := range []step{rateLimited, maintenance} {
		s := newScriptedServer(t, first, step{status: http.StatusConflict, body: `{"title":"Conflict"}`})
		c, _ := s.retryClient(t, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
		err := c.SubmitURL(context.Background(), testToken(), "scan-1", &URLSubmission{URL: "https://example.com", Properties: validSubmitProps})
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
			t.Errorf("after HTTP %d: err = %v, want the 409", first.status, err)
		}
	}
	// Without retries a 409 is always an error.
	env := newTestEnv(t, http.StatusConflict, "")
	if err := env.client(t).SubmitURL(context.Background(), testToken(), "scan-1", &URLSubmission{URL: "https://example.com", Properties: validSubmitProps}); err == nil {
		t.Error("err = nil, want the 409")
	}
}

func TestRetryStopsWhenTheContextEnds(t *testing.T) {
	s := newScriptedServer(t, rateLimited, ok200)
	c, _ := s.retryClient(t, RetryPolicy{MaxAttempts: 3})
	ctx, cancel := context.WithCancel(context.Background())
	sleep = func(context.Context, time.Duration) error {
		cancel()
		return context.Canceled
	}
	_, err := c.CreditsBalance(ctx, testToken())
	if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want context.Canceled and ErrRateLimited", err)
	}
	if n := len(s.requests()); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}

func TestRetryDoesNotWaitPastTheDeadline(t *testing.T) {
	s := newScriptedServer(t, step{status: http.StatusTooManyRequests, header: map[string]string{"Retry-After": "20"}}, ok200)
	c, slept := s.retryClient(t, RetryPolicy{MaxAttempts: 3})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.CreditsBalance(ctx, testToken())
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want context.DeadlineExceeded and ErrRateLimited", err)
	}
	if len(s.requests()) != 1 || len(*slept) != 0 {
		t.Errorf("%d requests and %d waits, want 1 and 0", len(s.requests()), len(*slept))
	}
}

func TestRetryWaitsForReal(t *testing.T) {
	s := newScriptedServer(t, step{status: http.StatusServiceUnavailable, header: map[string]string{"Retry-After": "1"}}, step{status: http.StatusOK, body: `{"Amount":5}`})
	c := mustClient(t, WithAPIBaseURL(s.srv.URL), WithRetry(RetryPolicy{MaxAttempts: 2}))
	start := time.Now()
	res, err := c.CreditsBalance(context.Background(), testToken())
	if err != nil || res.Amount != 5 {
		t.Fatalf("CreditsBalance = %+v, %v", res, err)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("returned after %v, want at least the 1s of Retry-After", elapsed)
	}
}

func TestAPIErrorRetryAfter(t *testing.T) {
	for header, want := range map[string]time.Duration{"": 0, "30": 30 * time.Second, "0": 0, "soon": 0, "-5": 0} {
		s := newScriptedServer(t, step{status: http.StatusTooManyRequests, header: map[string]string{"Retry-After": header}})
		_, err := mustClient(t, WithAPIBaseURL(s.srv.URL)).CreditsBalance(context.Background(), testToken())
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.RetryAfter != want {
			t.Errorf("Retry-After %q: err = %v, want RetryAfter %v", header, err, want)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	tests := map[string]time.Duration{
		"":                              0,
		"  ":                            0,
		"0":                             0,
		"5":                             5 * time.Second,
		" 12 ":                          12 * time.Second,
		"-1":                            0,
		"1.5":                           0,
		"999999999":                     24 * time.Hour,
		"Wed, 14 Oct 2026 12:01:00 GMT": time.Minute,
		"Wed, 14 Oct 2026 11:59:00 GMT": 0,
		"not a date":                    0,
	}
	for value, want := range tests {
		if got := parseRetryAfter(value, now); got != want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestRetryPolicyDelayBounds(t *testing.T) {
	p := RetryPolicy{BaseDelay: time.Second, MaxDelay: 10 * time.Second}
	for i := 0; i < 200; i++ {
		for attempt, max := range map[int]time.Duration{1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second, 4: 8 * time.Second, 5: 10 * time.Second, 40: 10 * time.Second} {
			d, ok := p.delay(attempt, 0)
			if !ok || d < max/2 || d > max {
				t.Fatalf("delay(%d) = %v, %v; want %v to %v", attempt, d, ok, max/2, max)
			}
		}
	}
	if d, ok := p.delay(1, 10*time.Second); !ok || d != 10*time.Second {
		t.Errorf("Retry-After at MaxDelay: %v, %v", d, ok)
	}
	if _, ok := p.delay(1, 11*time.Second); ok {
		t.Error("Retry-After above MaxDelay must stop the retries")
	}
}

func TestWithRetryOptions(t *testing.T) {
	c := mustClient(t, WithRetry(RetryPolicy{MaxAttempts: 3}))
	if c.retry != (RetryPolicy{MaxAttempts: 3, BaseDelay: time.Second, MaxDelay: 30 * time.Second}) {
		t.Errorf("defaults: %+v", c.retry)
	}
	c = mustClient(t, WithRetry(RetryPolicy{MaxAttempts: 2, BaseDelay: time.Minute, MaxDelay: time.Second}))
	if c.retry.BaseDelay != time.Second {
		t.Errorf("BaseDelay above MaxDelay: %+v", c.retry)
	}
	for _, p := range []RetryPolicy{{MaxAttempts: -1}, {BaseDelay: -1}, {MaxDelay: -1}} {
		if _, err := NewClient(WithRetry(p)); !errors.Is(err, ErrInvalidArgument) || !strings.Contains(err.Error(), "WithRetry") {
			t.Errorf("WithRetry(%+v): err = %v", p, err)
		}
	}
}

func equalDurations(a, b []time.Duration) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
