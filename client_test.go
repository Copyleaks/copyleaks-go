package copyleaks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
)

func TestAPIErrorMatchesSentinels(t *testing.T) {
	tests := []struct {
		status int
		target error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusTooManyRequests, ErrRateLimited},
		{http.StatusServiceUnavailable, ErrUnderMaintenance},
		{http.StatusBadRequest, nil},
		{http.StatusConflict, nil},
	}
	sentinels := []error{ErrUnauthorized, ErrRateLimited, ErrUnderMaintenance, ErrInvalidArgument, ErrTokenExpired}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			env := newTestEnv(t, tt.status, `{"message":"the server explains"}`)
			_, err := env.client(t).CreditsBalance(context.Background(), testToken())

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error %v is not an *APIError", err)
			}
			if apiErr.StatusCode != tt.status || apiErr.Method != http.MethodGet || apiErr.Endpoint != "/v3/scans/credits" {
				t.Errorf("APIError = %d %s %s", apiErr.StatusCode, apiErr.Method, apiErr.Endpoint)
			}
			if string(apiErr.Body) != `{"message":"the server explains"}` {
				t.Errorf("Body = %q", apiErr.Body)
			}
			for _, s := range sentinels {
				if got, want := errors.Is(err, s), s == tt.target; got != want {
					t.Errorf("errors.Is(err, %v) = %v, want %v", s, got, want)
				}
			}
			msg := err.Error()
			wantPrefix := fmt.Sprintf("copyleaks: GET /v3/scans/credits: HTTP %d %s: ", tt.status, http.StatusText(tt.status))
			if !strings.HasPrefix(msg, wantPrefix) || !strings.Contains(msg, "the server explains") {
				t.Errorf("Error() = %q, want the endpoint, the status and the body", msg)
			}
			if strings.Contains(msg, testAccessToken) {
				t.Errorf("Error() contains the access token: %q", msg)
			}
		})
	}
}

func TestAPIErrorBody(t *testing.T) {
	full := APIErrorBody{
		Type: ErrorTypeInvalidRequest, ID: "missing_parameter", Code: 1,
		Message: "Bad request. One or several required parameters are missing or incorrect.",
		URL:     "https://docs.copyleaks.com/using-the-apis/api-errors/#error-reference",
		Details: []APIErrorDetail{{Param: "labels", Message: "Id Label EXAMPLE-v1 is not supported"}},
	}
	tests := []struct {
		name string
		body string
		want APIErrorBody
	}{
		{"full envelope", `{"error":{"type":"invalid_request_error","id":"missing_parameter","code":1,` +
			`"message":"Bad request. One or several required parameters are missing or incorrect.",` +
			`"url":"https://docs.copyleaks.com/using-the-apis/api-errors/#error-reference",` +
			`"details":[{"param":"labels","message":"Id Label EXAMPLE-v1 is not supported"}]}}`, full},
		{"without the optional fields", `{"error":{"type":"payment_error","id":"insufficient_credits","code":7}}`,
			APIErrorBody{Type: ErrorTypePayment, ID: "insufficient_credits", Code: 7}},
		{"empty error object", `{"error":{}}`, APIErrorBody{}},
		{"code as a numeric string", `{"error":{"type":"api_error","code":"42"}}`, APIErrorBody{Type: ErrorTypeAPI, Code: 42}},
		{"code as a numeric string with whitespace", `{"error":{"code":" 7 "}}`, APIErrorBody{Code: 7}},
		{"code as a signed numeric string", `{"error":{"code":"-3"}}`, APIErrorBody{Code: -3}},
		{"code with inner whitespace", `{"error":{"code":"1 2"}}`, APIErrorBody{}},
		{"code only whitespace", `{"error":{"code":" "}}`, APIErrorBody{}},
		{"code not a number", `{"error":{"type":"authentication_error","code":"forty-two"}}`, APIErrorBody{Type: ErrorTypeAuthentication}},
		{"code not an integer", `{"error":{"code":1.5}}`, APIErrorBody{}},
		{"code an object", `{"error":{"code":{"value":1}}}`, APIErrorBody{}},
		{"fields of the wrong type", `{"error":{"type":1,"id":true,"message":["m"],"url":null,"code":null}}`, APIErrorBody{}},
		{"unknown type", `{"error":{"type":"future_error"}}`, APIErrorBody{Type: "future_error"}},
		{"not JSON", `<html>Bad Gateway</html>`, APIErrorBody{}},
		{"empty body", ``, APIErrorBody{}},
		{"JSON without error", `{"message":"the server explains"}`, APIErrorBody{}},
		{"error is a string", `{"error":"boom"}`, APIErrorBody{}},
		{"error is null", `{"error":null}`, APIErrorBody{}},
		{"JSON array", `[{"error":{"type":"api_error"}}]`, APIErrorBody{}},
		{"details not a list", `{"error":{"id":"x","details":{"param":"a"}}}`, APIErrorBody{ID: "x"}},
		{"details with bad entries", `{"error":{"details":[null,"text",1,{"param":2,"message":"m"},{},{"param":"p"}]}}`,
			APIErrorBody{Details: []APIErrorDetail{{Message: "m"}, {}, {Param: "p"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := &APIError{StatusCode: 400, Method: "POST", Endpoint: "/x", Body: []byte(tt.body)}
			if got := apiErr.ErrorBody(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ErrorBody() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAPIErrorBodyFromResponse(t *testing.T) {
	body := `{"error":{"type":"payment_error","id":"insufficient_credits","code":"7","message":"Not enough credits."}}`
	env := newTestEnv(t, http.StatusPaymentRequired, body)
	_, err := env.client(t).CreditsBalance(context.Background(), testToken())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error %v is not an *APIError", err)
	}
	want := APIErrorBody{Type: ErrorTypePayment, ID: "insufficient_credits", Code: 7, Message: "Not enough credits."}
	if got := apiErr.ErrorBody(); !reflect.DeepEqual(got, want) {
		t.Errorf("ErrorBody() = %+v, want %+v", got, want)
	}
	// The message and the raw body stay as they were.
	if string(apiErr.Body) != body || err.Error() != "copyleaks: GET /v3/scans/credits: HTTP 402 Payment Required: "+body {
		t.Errorf("err = %q, Body = %q", err, apiErr.Body)
	}
}

func TestAPIErrorTruncatesLongBody(t *testing.T) {
	err := &APIError{StatusCode: 400, Method: "PUT", Endpoint: "/x", Body: []byte(strings.Repeat("é", 400))}
	msg := err.Error()
	if !strings.HasSuffix(msg, "...") || len(msg) > 600 || !strings.Contains(msg, "HTTP 400 Bad Request") {
		t.Errorf("Error() = %q", msg)
	}
	if !utf8.ValidString(msg) {
		t.Errorf("Error() splits a UTF-8 sequence: %q", msg)
	}
}

func TestLoginErrorDoesNotLeakTheKey(t *testing.T) {
	env := newTestEnv(t, http.StatusUnauthorized, `{"message":"Invalid email or API key"}`)
	_, err := env.client(t).Login(context.Background(), "user@example.com", "secret-api-key")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if strings.Contains(err.Error(), "secret-api-key") {
		t.Errorf("Error() contains the API key: %q", err)
	}
}

func TestNetworkErrorDoesNotLeakTheToken(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	c := mustClient(t, WithAPIBaseURL(srv.URL))
	_, err := c.CreditsBalance(context.Background(), testToken())
	if err == nil {
		t.Fatal("expected an error from a closed server")
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Errorf("a network error must not be an *APIError: %v", err)
	}
	if !strings.HasPrefix(err.Error(), "copyleaks: GET /v3/scans/credits: ") || strings.Contains(err.Error(), testAccessToken) {
		t.Errorf("Error() = %q", err)
	}
}

func TestCanceledContext(t *testing.T) {
	env := newTestEnv(t, http.StatusOK, `["en"]`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := env.client(t).OCRSupportedLanguages(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestDecodeErrorOnMalformedResponse(t *testing.T) {
	env := newTestEnv(t, http.StatusOK, `{"Amount":`)
	_, err := env.client(t).CreditsBalance(context.Background(), testToken())
	if err == nil || !strings.Contains(err.Error(), "copyleaks: decode GET /v3/scans/credits response") {
		t.Errorf("err = %v, want a decode error", err)
	}
}

func TestUserAgent(t *testing.T) {
	if sdkUserAgent != "copyleaks-go-sdk/"+Version {
		t.Fatalf("sdkUserAgent = %q", sdkUserAgent)
	}
	tests := []struct {
		name    string
		options []Option
		want    string
	}{
		{"default", nil, "copyleaks-go-sdk/" + Version},
		{"application token", []Option{WithUserAgent("my-app/2.1")}, "my-app/2.1 copyleaks-go-sdk/" + Version},
		{"blank is ignored", []Option{WithUserAgent("  ")}, "copyleaks-go-sdk/" + Version},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, `["en"]`)
			if _, err := env.client(t, tt.options...).OCRSupportedLanguages(context.Background()); err != nil {
				t.Fatal(err)
			}
			if got := env.next(t).header.Get("User-Agent"); got != tt.want {
				t.Errorf("User-Agent = %q, want %q", got, tt.want)
			}
		})
	}
}

// countingTransport counts the requests that pass through it.
type countingTransport struct {
	count int
	// base sends the requests; nil means http.DefaultTransport.
	base http.RoundTripper
}

func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.count++
	if c.base != nil {
		return c.base.RoundTrip(r)
	}
	return http.DefaultTransport.RoundTrip(r)
}

func TestWithHTTPClient(t *testing.T) {
	env := newTestEnv(t, http.StatusOK, `["en"]`)
	transport := &countingTransport{}
	c := env.client(t, WithHTTPClient(&http.Client{Transport: transport}), WithHTTPClient(nil))
	if _, err := c.OCRSupportedLanguages(context.Background()); err != nil {
		t.Fatal(err)
	}
	if transport.count != 1 {
		t.Errorf("custom http.Client sent %d requests, want 1", transport.count)
	}
}

func TestNoRetries(t *testing.T) {
	env := newTestEnv(t, http.StatusServiceUnavailable, "maintenance")
	_, err := env.client(t).OCRSupportedLanguages(context.Background())
	if !errors.Is(err, ErrUnderMaintenance) {
		t.Fatalf("err = %v, want ErrUnderMaintenance", err)
	}
	env.next(t)
	env.none(t)
}

func TestBaseURLTrailingSlash(t *testing.T) {
	env := newTestEnv(t, http.StatusOK, `["en"]`)
	c := mustClient(t, WithAPIBaseURL(env.api.URL+"/"))
	if _, err := c.OCRSupportedLanguages(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := env.next(t).path; got != "/v3/miscellaneous/ocr-languages-list" {
		t.Errorf("path = %q", got)
	}
}

func TestDefaultBaseURLs(t *testing.T) {
	c := mustClient(t)
	if c.apiBaseURL != "https://api.copyleaks.com" || c.identityBaseURL != "https://id.copyleaks.com" {
		t.Errorf("base URLs = %q, %q", c.apiBaseURL, c.identityBaseURL)
	}
	if c.httpClient == nil || c.httpClient.Timeout != 2*time.Minute {
		t.Errorf("default http.Client = %+v", c.httpClient)
	}
}

func TestNilClientAndNilContextDoNotPanic(t *testing.T) {
	var nilClient *Client
	if _, err := nilClient.ReleaseNotes(context.Background()); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("nil client: err = %v, want ErrInvalidArgument", err)
	}
	var nilCtx context.Context
	if _, err := mustClient(t).ReleaseNotes(nilCtx); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("nil context: err = %v, want ErrInvalidArgument", err)
	}
}

func TestResponseBodyCaps(t *testing.T) {
	oldJSON, oldCSV := maxJSONBody, maxCSVBody
	t.Cleanup(func() { maxJSONBody, maxCSVBody = oldJSON, oldCSV })
	maxJSONBody, maxCSVBody = 16, 8

	ctx := context.Background()
	day := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	credits := func(c *Client) error {
		_, err := c.CreditsBalance(ctx, testToken())
		return err
	}
	usages := func(c *Client) error {
		_, err := c.UsagesHistoryCSV(ctx, testToken(), day, day)
		return err
	}
	tests := []struct {
		name string
		body string
		call func(c *Client) error
		ok   bool
	}{
		{"JSON at the cap", `{"Amount":12345}`, credits, true},
		{"JSON over the cap", `{"Amount":123456}`, credits, false},
		{"CSV at the cap", "a,b\n1,2\n", usages, true},
		{"CSV over the cap", "a,b\n1,2\n3", usages, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, tt.body)
			err := tt.call(env.client(t))
			if tt.ok {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, ErrResponseTooLarge) || !strings.Contains(err.Error(), "larger than") {
				t.Errorf("err = %v, want ErrResponseTooLarge", err)
			}
		})
	}
}

func TestConnectionIsReused(t *testing.T) {
	var conns atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/scans/credits":
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, strings.Repeat("x", 2000))
		case "/v3/miscellaneous/ocr-languages-list":
			_, _ = io.WriteString(w, `["en"]`+strings.Repeat(" ", 2000))
		default:
			_, _ = io.WriteString(w, strings.Repeat("y", 2000))
		}
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	c := mustClient(t, WithAPIBaseURL(srv.URL))
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := c.OCRSupportedLanguages(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := c.CreditsBalance(ctx, testToken()); !errors.Is(err, ErrRateLimited) {
			t.Fatalf("err = %v, want ErrRateLimited", err)
		}
		if err := c.ResendWebhook(ctx, testToken(), "scan-1"); err != nil {
			t.Fatal(err)
		}
	}
	if n := conns.Load(); n != 1 {
		t.Errorf("9 calls opened %d connections, want 1", n)
	}
}

func TestBaseURLMustUseHTTPS(t *testing.T) {
	tests := []struct {
		url string
		ok  bool
	}{
		{"https://api.copyleaks.com", true},
		{"https://api.copyleaks.com/", true},
		{"http://localhost:8080", true},
		{"http://LOCALHOST", true},
		{"http://127.0.0.1:1234", true},
		{"http://[::1]:1234", true},
		{"http://api.copyleaks.com", false},
		{"http://127.0.0.2", false},
		{"http://user:secret@example.com", false},
		{"ftp://api.copyleaks.com", false},
		{"api.copyleaks.com", false},
		{"", false},
	}
	options := map[string]func(string) Option{"WithAPIBaseURL": WithAPIBaseURL, "WithIdentityBaseURL": WithIdentityBaseURL}
	for _, tt := range tests {
		for name, option := range options {
			t.Run(name+"/"+tt.url, func(t *testing.T) {
				c, err := NewClient(option(tt.url))
				if tt.ok {
					if err != nil || c == nil {
						t.Fatalf("NewClient = %v, %v; want a client", c, err)
					}
					return
				}
				if c != nil || !errors.Is(err, ErrInvalidArgument) {
					t.Fatalf("NewClient = %v, %v; want ErrInvalidArgument", c, err)
				}
				if !strings.Contains(err.Error(), name) || strings.Contains(err.Error(), "secret") {
					t.Errorf("Error() = %q", err)
				}
			})
		}
	}
}
