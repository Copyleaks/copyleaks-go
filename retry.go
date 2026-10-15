package copyleaks

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Defaults of RetryPolicy.
const (
	defaultRetryBaseDelay = time.Second
	defaultRetryMaxDelay  = 30 * time.Second
)

// RetryPolicy configures WithRetry. Only calls that are safe to send twice are retried: every GET, Login, the
// submits that carry your scan ID (SubmitFile, SubmitOCR, SubmitURL, SubmitAIImageAsync, SubmitAIVideo; the
// server rejects a duplicate ID) and Export, except after HTTP 503 (a failed export keeps its ID, so a retry
// would get 409). Calls that charge credits per request (DetectAIText, DetectAIImage, CheckGrammar, ModerateText,
// CheckImagePlagiarism), Start, Delete and ResendWebhook are never retried; their *APIError still has RetryAfter.
//
// Retries follow HTTP 429, HTTP 503 and connection errors (timeout, refused, reset or dropped connection, HTTP/2
// shutdown), not TLS or unknown-host errors. Each waits for Retry-After, or else for exponential backoff with
// jitter from BaseDelay up to MaxDelay. A 409 after an attempt the server may have processed counts as success.
type RetryPolicy struct {
	// MaxAttempts counts the attempts of each call, the first included; below 2 turns retries off.
	MaxAttempts int
	// BaseDelay is the delay before the first retry, before jitter (default one second).
	BaseDelay time.Duration
	// MaxDelay caps each delay (default 30 seconds); a longer Retry-After ends the call with its error.
	MaxDelay time.Duration
}

// WithRetry turns on retries of transient failures (off by default). The context bounds the whole call, retries
// included; a delay that ends after its deadline is not waited for. Negative fields give ErrInvalidArgument.
func WithRetry(policy RetryPolicy) Option {
	return func(c *Client) error {
		if policy.MaxAttempts < 0 || policy.BaseDelay < 0 || policy.MaxDelay < 0 {
			return invalid("WithRetry needs a RetryPolicy without negative fields")
		}
		if policy.BaseDelay == 0 {
			policy.BaseDelay = defaultRetryBaseDelay
		}
		if policy.MaxDelay == 0 {
			policy.MaxDelay = defaultRetryMaxDelay
		}
		if policy.BaseDelay > policy.MaxDelay {
			policy.BaseDelay = policy.MaxDelay
		}
		c.retry = policy
		return nil
	}
}

// delay returns the wait before the retry after attempt (1-based), or false if retryAfter exceeds MaxDelay.
func (p RetryPolicy) delay(attempt int, retryAfter time.Duration) (time.Duration, bool) {
	if retryAfter > 0 {
		return retryAfter, retryAfter <= p.MaxDelay
	}
	d := p.BaseDelay
	for i := 1; i < attempt && d < p.MaxDelay; i++ {
		d *= 2
	}
	if d > p.MaxDelay {
		d = p.MaxDelay
	}
	// Equal jitter, so that clients that failed together do not retry together.
	half := d / 2
	return half + rand.N(d-half+1), true
}

// parseRetryAfter parses a Retry-After value (seconds or HTTP date); 0 for an empty, invalid or past value.
func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if secs, err := strconv.ParseInt(value, 10, 64); err == nil {
		if secs <= 0 {
			return 0
		}
		if secs > int64(24*time.Hour/time.Second) {
			secs = int64(24 * time.Hour / time.Second)
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil && t.After(now) {
		return t.Sub(now)
	}
	return 0
}

// sleep is a variable so that tests can replace it.
var sleep = sleepContext

// sleepContext waits for d, or returns ctx.Err() when ctx is done first.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
