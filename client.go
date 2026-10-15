package copyleaks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Copyleaks/copyleaks-go/internal/netretry"
	"github.com/Copyleaks/copyleaks-go/internal/upload"
)

const (
	// DefaultAPIBaseURL is the base URL of the Copyleaks API.
	DefaultAPIBaseURL = "https://api.copyleaks.com"
	// DefaultIdentityBaseURL is the base URL of the Copyleaks identity server, used by Login.
	DefaultIdentityBaseURL = "https://id.copyleaks.com"
)

// defaultTimeout bounds each request unless WithHTTPClient or WithTimeout is used.
const defaultTimeout = 2 * time.Minute

// maxErrorBody caps how much of an error response body is kept in an APIError.
const maxErrorBody = 1 << 20

// maxDrain caps how much of an unread body is discarded on close, so the connection can be reused.
const maxDrain = 64 << 10

// maxCSVBody caps a CSV response. It is a variable so that tests can lower it.
var maxCSVBody int64 = 64 << 20

// maxJSONBody is the default JSON response cap (see WithMaxJSONResponseSize), a variable so tests can lower it.
var maxJSONBody int64 = 16 << 20

// Client calls the Copyleaks API. It is safe for concurrent use: create one with NewClient and reuse it.
// It holds no credentials; pass the AuthToken from Login to each authenticated call.
type Client struct {
	httpClient      *http.Client
	apiBaseURL      string
	identityBaseURL string
	userAgent       string
	retry           RetryPolicy
	maxJSON         int64
	timeout         time.Duration
}

// Option configures a Client. Pass options to NewClient.
type Option func(*Client) error

// NewClient returns a Client for the production API; an invalid option gives ErrInvalidArgument.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		apiBaseURL:      DefaultAPIBaseURL,
		identityBaseURL: DefaultIdentityBaseURL,
		userAgent:       sdkUserAgent,
		timeout:         defaultTimeout,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	if c.httpClient == nil {
		// No Transport of its own: each request uses http.DefaultTransport as it is at send time.
		c.httpClient = &http.Client{Timeout: c.timeout}
	}
	return c, nil
}

// WithHTTPClient sends requests with hc instead of the default http.Client, e.g. for a proxy or custom
// Transport. hc is used as is and never changed; nil is ignored.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc != nil {
			c.httpClient = hc
		}
		return nil
	}
}

// WithTimeout sets the per-request timeout of the default http.Client, up to the end of the response body
// (default two minutes; 0 means none, so only the context bounds it). Each WithRetry attempt gets the full
// timeout. It has no effect with WithHTTPClient. A negative timeout gives ErrInvalidArgument.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		if timeout < 0 {
			return invalid("WithTimeout needs a timeout of 0 or more")
		}
		c.timeout = timeout
		return nil
	}
}

// WithMaxJSONResponseSize sets the largest JSON response body accepted, in bytes (default 16 MiB, minimum 1).
// A larger body gives ErrResponseTooLarge; raise it for the AI Image Detection masks of large images.
func WithMaxJSONResponseSize(size int64) Option {
	return func(c *Client) error {
		if size < 1 {
			return invalid("WithMaxJSONResponseSize needs a size of at least 1 byte")
		}
		c.maxJSON = size
		return nil
	}
}

// WithAPIBaseURL replaces DefaultAPIBaseURL. It must use https; http is only accepted for localhost,
// 127.0.0.1 and ::1.
func WithAPIBaseURL(baseURL string) Option {
	return func(c *Client) error {
		u, err := checkBaseURL("WithAPIBaseURL", baseURL)
		if err != nil {
			return err
		}
		c.apiBaseURL = u
		return nil
	}
}

// WithIdentityBaseURL replaces DefaultIdentityBaseURL, with the same https rule as WithAPIBaseURL.
func WithIdentityBaseURL(baseURL string) Option {
	return func(c *Client) error {
		u, err := checkBaseURL("WithIdentityBaseURL", baseURL)
		if err != nil {
			return err
		}
		c.identityBaseURL = u
		return nil
	}
}

// checkBaseURL returns baseURL without trailing slashes if it is https, or http on a loopback host.
// The error names option but not the URL, which may hold credentials.
func checkBaseURL(option, baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return "", invalid(option + " needs an absolute URL, such as https://api.copyleaks.com")
	}
	host := u.Hostname()
	loopback := strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "::1"
	if u.Scheme != "https" && !(u.Scheme == "http" && loopback) {
		return "", invalid(option + " needs an https URL; http is only accepted for localhost, 127.0.0.1 and ::1")
	}
	return strings.TrimRight(baseURL, "/"), nil
}

// WithUserAgent prepends an application token, such as "my-app/2.1", to the SDK's "copyleaks-go-sdk/<Version>".
func WithUserAgent(userAgent string) Option {
	return func(c *Client) error {
		if ua := strings.TrimSpace(userAgent); ua != "" {
			c.userAgent = ua + " " + sdkUserAgent
		}
		return nil
	}
}

// apiRequest describes one HTTP call. path holds escaped segments; identity targets the identity server; token
// is nil for public endpoints. body is sent as JSON, rawBody as is with contentType. idempotent marks a call
// that is safe to resend (GET always is); noRetryOn503 stops its retries after HTTP 503.
type apiRequest struct {
	method       string
	path         string
	query        url.Values
	identity     bool
	token        *AuthToken
	body         any
	rawBody      []byte
	contentType  string
	idempotent   bool
	noRetryOn503 bool
}

func (r *apiRequest) retryable() bool {
	return r.idempotent || r.method == http.MethodGet
}

// encodeBody returns the request body and its content type, or nil when there is none.
func (r *apiRequest) encodeBody() ([]byte, string, error) {
	if r.rawBody != nil {
		return r.rawBody, r.contentType, nil
	}
	if r.body == nil {
		return nil, "", nil
	}
	b, err := encodeJSON(r.body)
	if err != nil {
		return nil, "", fmt.Errorf("copyleaks: encode %s %s request: %w", r.method, r.path, err)
	}
	return b, "application/json", nil
}

// encodeJSON is json.Marshal without escaping <, > and &, which only makes HTML or code text larger.
func encodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	return b[:len(b)-1], nil // Encode ends the value with a newline.
}

// base64Body is upload.Base64Body for v, a request whose first field, Base64, is set to "".
func base64Body(v any, b64 string, raw []byte) ([]byte, error) {
	head, err := encodeJSON(v)
	if err != nil {
		return nil, err
	}
	return upload.Base64Body(head, b64, raw)
}

// attemptResult is the outcome of one attempt of a request.
type attemptResult struct {
	// err is nil on success.
	err error
	// status is the HTTP status, or 0 when no response arrived.
	status int
	// retryAfter is the Retry-After delay of the response.
	retryAfter time.Duration
	// transient reports a failure that a retry may fix: HTTP 429 or 503, or see netretry.Transient.
	transient bool
	// maybeProcessed reports a network error after the whole request was sent: the server may have processed it.
	maybeProcessed bool
}

// do sends r, retrying per WithRetry, and decodes a 2xx body into out: discarded for nil, copied for *[]byte
// (maxCSVBody), JSON otherwise (maxJSON). A non-2xx response becomes an *APIError.
func (c *Client) do(ctx context.Context, r apiRequest, out any) error {
	if c == nil {
		return invalid("nil *Client; create one with NewClient")
	}
	if ctx == nil {
		return invalid("nil context")
	}
	body, contentType, err := r.encodeBody()
	if err != nil {
		return err
	}
	base := c.apiBaseURL
	if r.identity {
		base = c.identityBaseURL
	}
	target := base + r.path
	if len(r.query) > 0 {
		target += "?" + r.query.Encode()
	}

	attempts := 1
	if r.retryable() && c.retry.MaxAttempts > 1 {
		attempts = c.retry.MaxAttempts
	}
	maybeDone := false
	for attempt := 1; ; attempt++ {
		res := c.attempt(ctx, r, target, body, contentType, out)
		if res.err == nil {
			return nil
		}
		// A submit or export whose ID exists answers 409: after an attempt that the server may have
		// processed, it means that attempt succeeded.
		if res.status == http.StatusConflict && maybeDone && out == nil {
			return nil
		}
		if attempt >= attempts || !res.transient || (r.noRetryOn503 && res.status == http.StatusServiceUnavailable) {
			return res.err
		}
		maybeDone = maybeDone || res.maybeProcessed
		delay, ok := c.retry.delay(attempt, res.retryAfter)
		if !ok {
			return res.err
		}
		if deadline, has := ctx.Deadline(); has && time.Until(deadline) < delay {
			return fmt.Errorf("%w (not retried: %w)", res.err, context.DeadlineExceeded)
		}
		if err := sleep(ctx, delay); err != nil {
			return fmt.Errorf("%w (not retried: %w)", res.err, err)
		}
	}
}

// attempt sends r once.
func (c *Client) attempt(ctx context.Context, r apiRequest, target string, body []byte, contentType string, out any) attemptResult {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	// wrote records that the whole request was sent; the transport may set it from another goroutine.
	var wrote atomic.Bool
	traced := httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				wrote.Store(true)
			}
		},
	})
	req, err := http.NewRequestWithContext(traced, r.method, target, bodyReader)
	if err != nil {
		return attemptResult{err: fmt.Errorf("copyleaks: build %s %s request: %w", r.method, r.path, err)}
	}
	req.Header.Set("User-Agent", c.userAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if r.token != nil {
		req.Header.Set("Authorization", "Bearer "+r.token.AccessToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return attemptResult{
			err:            fmt.Errorf("copyleaks: %s %s: %w", r.method, r.path, err),
			transient:      ctx.Err() == nil && netretry.Transient(err),
			maybeProcessed: wrote.Load(),
		}
	}
	defer func() {
		// Discard what is left of the body, so that the connection can be reused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrain))
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
		return attemptResult{
			err:        &APIError{StatusCode: resp.StatusCode, Method: r.method, Endpoint: r.path, Body: b, RetryAfter: retryAfter},
			status:     resp.StatusCode,
			retryAfter: retryAfter,
			transient:  resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable,
		}
	}
	// A connection error while reading a 2xx body is transient.
	readFailed := func(err error) attemptResult {
		return attemptResult{
			err:            fmt.Errorf("copyleaks: read %s %s response: %w", r.method, r.path, err),
			transient:      ctx.Err() == nil && netretry.Transient(err),
			maybeProcessed: true,
		}
	}
	switch v := out.(type) {
	case nil:
		return attemptResult{}
	case *[]byte:
		b, err := readCapped(resp.Body, maxCSVBody)
		if err != nil {
			return readFailed(err)
		}
		*v = b
		return attemptResult{}
	default:
		limit := c.maxJSON
		if limit <= 0 {
			limit = maxJSONBody
		}
		b, err := readCapped(resp.Body, limit)
		if err != nil {
			return readFailed(err)
		}
		if err := json.Unmarshal(b, out); err != nil {
			return attemptResult{err: fmt.Errorf("copyleaks: decode %s %s response: %w", r.method, r.path, err)}
		}
		return attemptResult{}
	}
}

// readCapped reads all of r, or fails with ErrResponseTooLarge and no data past limit bytes.
func readCapped(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("%w: the body is larger than %d bytes", ErrResponseTooLarge, limit)
	}
	return b, nil
}

// segment escapes s as one URL path segment.
func segment(s string) string {
	return url.PathEscape(s)
}
