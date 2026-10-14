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

// RetryPolicy configures the retries that WithRetry turns on.
//
// The SDK only retries calls that are safe to send twice:
//   - every GET call, such as CreditsBalance, UsagesHistoryCSV and the lookups;
//   - Login;
//   - the submits that carry your scan ID: SubmitFile, SubmitOCR, SubmitURL,
//     SubmitAIImageAsync and SubmitAIVideo. The server rejects a second scan with the
//     same ID, so a retry never starts a second scan;
//   - Export, which the server also rejects for an export ID that already exists.
//
// It never retries the calls that charge credits for each request, such as DetectAIText,
// DetectAIImage, CheckGrammar, ModerateText and CheckImagePlagiarism, nor Start,
// Delete and ResendWebhook. Their errors still report the server's Retry-After delay in
// APIError.RetryAfter.
//
// A call is retried after HTTP 429 (rate limited), HTTP 503 (under maintenance) and
// network errors, such as a refused or dropped connection. Each retry waits for the
// Retry-After delay of the response when it has one, and otherwise for an exponential
// backoff with jitter: about BaseDelay, then twice that, and so on, up to MaxDelay.
//
// When an attempt of a submit or an export fails with a network error after the whole
// request was sent, the server may have processed it. If a later attempt then gets HTTP
// 409 Conflict, the call counts as a success, as the scan or the export exists.
type RetryPolicy struct {
	// MaxAttempts is the number of attempts of each call, including the first one.
	// Values below 2 turn retries off.
	MaxAttempts int
	// BaseDelay is the delay before the first retry, before jitter. Zero means one
	// second.
	BaseDelay time.Duration
	// MaxDelay caps each delay. Zero means 30 seconds. When a response asks for a longer
	// Retry-After delay, the call is not retried and returns that response's error.
	MaxDelay time.Duration
}

// WithRetry turns on retries of transient failures, as policy describes. Retries are
// off by default. The context passed to each call bounds the whole call, retries
// included: a call stops waiting when the context is done, and does not wait for a
// delay that ends after the context's deadline.
//
//	client, err := copyleaks.NewClient(copyleaks.WithRetry(copyleaks.RetryPolicy{MaxAttempts: 4}))
//
// A negative field gives an error that matches ErrInvalidArgument.
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

// delay returns how long to wait before the retry that follows attempt (1 for the first
// attempt), and false when the call must not be retried because retryAfter is longer
// than MaxDelay.
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
	// Equal jitter: half of the delay, plus a random part of the other half, so that
	// clients that failed together do not retry together.
	half := d / 2
	return half + rand.N(d-half+1), true
}

// parseRetryAfter returns the delay of a Retry-After header, which holds either a number
// of seconds or an HTTP date. It returns 0 for an empty, invalid or past value.
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

// sleepContext waits for d, or until ctx is done, and then returns ctx.Err().
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
