package copyleaks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// DefaultAPIBaseURL is the base URL of the Copyleaks API.
	DefaultAPIBaseURL = "https://api.copyleaks.com"
	// DefaultIdentityBaseURL is the base URL of the Copyleaks identity server, used by Login.
	DefaultIdentityBaseURL = "https://id.copyleaks.com"
)

// defaultTimeout bounds each request when no http.Client is set with WithHTTPClient and
// no other timeout is set with WithTimeout.
const defaultTimeout = 2 * time.Minute

// defaultMaxIdleConnsPerHost is how many idle connections to each host the default
// transport keeps, so that bursts of concurrent calls over HTTP/1.1 reuse connections.
const defaultMaxIdleConnsPerHost = 32

// maxErrorBody caps how much of an error response body is kept in an APIError.
const maxErrorBody = 1 << 20

// maxDrain caps how much of an unread response body is discarded before the body is
// closed, so that the connection can be reused.
const maxDrain = 64 << 10

// maxCSVBody caps the size of a CSV response. It is a variable so that tests can lower it.
var maxCSVBody int64 = 64 << 20

// maxJSONBody is the default cap on the size of a JSON response. It is a variable so that
// tests can lower it. WithMaxJSONResponseSize overrides it for one Client.
var maxJSONBody int64 = 16 << 20

// Client calls the Copyleaks API.
//
// A Client is safe for concurrent use by multiple goroutines. Create one with NewClient
// and reuse it. A Client holds no credentials: pass the AuthToken returned by Login to
// each authenticated call.
type Client struct {
	httpClient      *http.Client
	apiBaseURL      string
	identityBaseURL string
	userAgent       string
	retry           RetryPolicy
	maxJSON         int64
	timeout         time.Duration
	// sleep waits for d or until ctx is done. Tests replace it.
	sleep func(ctx context.Context, d time.Duration) error
}

// Option configures a Client. Pass options to NewClient.
type Option func(*Client) error

// NewClient returns a Client for the production Copyleaks API, configured by opts. It
// returns an error that matches ErrInvalidArgument when an option is invalid, such as a
// base URL that does not use https.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		apiBaseURL:      DefaultAPIBaseURL,
		identityBaseURL: DefaultIdentityBaseURL,
		userAgent:       sdkUserAgent,
		timeout:         defaultTimeout,
		sleep:           sleepContext,
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
		c.httpClient = &http.Client{Timeout: c.timeout, Transport: defaultTransport()}
	}
	return c, nil
}

// defaultTransport returns the transport of the default http.Client: a copy of
// http.DefaultTransport that keeps more idle connections per host. It is a copy, so
// that changes to http.DefaultTransport after NewClient do not affect the Client, and
// the SDK never changes http.DefaultTransport.
func defaultTransport() http.RoundTripper {
	var t *http.Transport
	if dt, ok := http.DefaultTransport.(*http.Transport); ok {
		t = dt.Clone()
	} else {
		t = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}
	t.MaxIdleConnsPerHost = defaultMaxIdleConnsPerHost
	return t
}

// WithHTTPClient makes the Client send its requests with hc instead of the default
// http.Client, which has a two-minute timeout (see WithTimeout). Use it to set a proxy
// or a custom Transport. The SDK uses hc as it is and never changes it. A nil hc is
// ignored.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc != nil {
			c.httpClient = hc
		}
		return nil
	}
}

// WithTimeout sets the timeout of each request of the default http.Client, from
// connecting to reading the whole response. The default is two minutes. Zero means no
// timeout: the context passed to each call then bounds it. Raise it to upload large
// files over slow links. With WithRetry, each attempt gets the full timeout.
//
// It has no effect when WithHTTPClient is also used: set the timeout on that client. A
// negative timeout gives an error that matches ErrInvalidArgument.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) error {
		if timeout < 0 {
			return invalid("WithTimeout needs a timeout of 0 or more")
		}
		c.timeout = timeout
		return nil
	}
}

// WithMaxJSONResponseSize sets the largest successful JSON response body that the
// Client accepts, in bytes. The default is 16 MiB. A larger body gives an error that
// matches ErrResponseTooLarge. Raise it for very large responses, such as the AI Image
// Detection masks of large, detailed images. A size below 1 gives an error that
// matches ErrInvalidArgument.
func WithMaxJSONResponseSize(size int64) Option {
	return func(c *Client) error {
		if size < 1 {
			return invalid("WithMaxJSONResponseSize needs a size of at least 1 byte")
		}
		c.maxJSON = size
		return nil
	}
}

// WithAPIBaseURL replaces DefaultAPIBaseURL. The URL must use https; plain http is only
// accepted for the hosts localhost, 127.0.0.1 and ::1, for local test servers.
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

// WithIdentityBaseURL replaces DefaultIdentityBaseURL, the server that Login calls. The
// URL must use https; plain http is only accepted for the hosts localhost, 127.0.0.1 and
// ::1, for local test servers.
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

// checkBaseURL returns baseURL without trailing slashes when it is an absolute https URL,
// or an http URL of a loopback host. option names the option in the error, which does
// not repeat the URL, as it may hold credentials.
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

// WithUserAgent adds an application product token, such as "my-app/2.1", in front of the
// SDK's own User-Agent token "copyleaks-go-sdk/<Version>", which is always sent.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) error {
		if ua := strings.TrimSpace(userAgent); ua != "" {
			c.userAgent = ua + " " + sdkUserAgent
		}
		return nil
	}
}

// apiRequest describes one HTTP call. path holds already escaped segments. identity
// selects the identity server instead of the API server. token is nil for public
// endpoints. body, when not nil, is sent as JSON; rawBody, when not nil, is sent as it
// is with the contentType header. idempotent marks a call that is safe to send again
// (see WithRetry); GET calls always are.
type apiRequest struct {
	method      string
	path        string
	query       url.Values
	identity    bool
	token       *AuthToken
	body        any
	rawBody     []byte
	contentType string
	idempotent  bool
}

// retryable reports whether r may be retried.
func (r *apiRequest) retryable() bool {
	return r.idempotent || r.method == http.MethodGet
}

// encodeBody returns the bytes and the content type of the request body, or nil when
// the request has no body.
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

// encodeJSON returns the JSON encoding of v. Unlike json.Marshal, it does not escape
// <, > and &, which would make text with HTML or code larger for no benefit.
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

// attemptResult is the outcome of one attempt of a request.
type attemptResult struct {
	// err is nil on success.
	err error
	// status is the HTTP status of the response, or 0 when no response arrived.
	status int
	// retryAfter is the delay that the Retry-After header of the response asked for.
	retryAfter time.Duration
	// transient reports a failure that a retry may fix: HTTP 429 or 503, or a network
	// error.
	transient bool
	// maybeProcessed reports a network error after the whole request was sent, so the
	// server may have processed it.
	maybeProcessed bool
}

// do sends r and decodes a successful response into out: the body is discarded when out
// is nil, copied when out is a *[]byte (up to 64 MiB), and decoded as JSON otherwise (up
// to 16 MiB, or the WithMaxJSONResponseSize limit). A larger body gives an error that
// matches ErrResponseTooLarge. A response with a non-2xx status becomes an *APIError.
//
// With WithRetry, a retryable request is sent again after a transient failure.
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
		// A submit whose scan ID, or an export whose export ID, already exists answers
		// 409. After an attempt that the server may have processed, that 409 means
		// that the earlier attempt succeeded.
		if res.status == http.StatusConflict && maybeDone && out == nil {
			return nil
		}
		if attempt >= attempts || !res.transient {
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
		if err := c.sleep(ctx, delay); err != nil {
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
	// wrote records that the whole request was sent. The transport may report it from
	// another goroutine.
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
			transient:      ctx.Err() == nil,
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
	// readFailed is the result of a failure to read a successful response body. A
	// network error while reading it is transient.
	readFailed := func(err error) attemptResult {
		return attemptResult{
			err:            fmt.Errorf("copyleaks: read %s %s response: %w", r.method, r.path, err),
			transient:      !errors.Is(err, ErrResponseTooLarge) && ctx.Err() == nil,
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

// readCapped reads all of r. It returns an error that matches ErrResponseTooLarge, and
// no data, when r holds more than limit bytes.
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

// segment escapes a value for use as one URL path segment.
func segment(s string) string {
	return url.PathEscape(s)
}
