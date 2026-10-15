package copyleaks

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	// ErrUnauthorized matches an *APIError with HTTP 401: the login token or the API key was rejected.
	ErrUnauthorized = errors.New("copyleaks: unauthorized")
	// ErrRateLimited matches an *APIError with HTTP 429. Wait before calling again (see APIError.RetryAfter).
	ErrRateLimited = errors.New("copyleaks: rate limited")
	// ErrUnderMaintenance matches an *APIError with HTTP 503 (maintenance); retry later, or use WithRetry.
	ErrUnderMaintenance = errors.New("copyleaks: under maintenance")
	// ErrTokenExpired is returned, before any request, when the AuthToken expired or expires within five minutes.
	ErrTokenExpired = errors.New("copyleaks: auth token expired")
	// ErrInvalidArgument matches the client-side checks that fail before any request: a missing required field,
	// a malformed scan ID, a too-short text, a non-https base URL, and so on.
	ErrInvalidArgument = errors.New("copyleaks: invalid argument")
	// ErrResponseTooLarge matches a 2xx body over 64 MiB (usage CSV) or 16 MiB (JSON); it is not truncated.
	ErrResponseTooLarge = errors.New("copyleaks: response too large")
)

// maxErrorMessageBody caps how much of the response body APIError.Error includes.
const maxErrorMessageBody = 512

// APIError is returned for a non-2xx HTTP status. Match it with errors.Is (ErrUnauthorized, ErrRateLimited,
// ErrUnderMaintenance) or read it with errors.As.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// Method is the HTTP method of the request.
	Method string
	// Endpoint is the URL path of the request, such as "/v3/scans/submit/file/my-scan".
	Endpoint string
	// Body is the response body, up to 1 MiB. The server usually explains the error here.
	Body []byte
	// RetryAfter is the Retry-After delay of the response, or 0 without one; mostly set on HTTP 429 and 503.
	RetryAfter time.Duration
}

// Error returns the method, endpoint, status and the start of the response body.
func (e *APIError) Error() string {
	status := strconv.Itoa(e.StatusCode)
	if text := http.StatusText(e.StatusCode); text != "" {
		status += " " + text
	}
	msg := fmt.Sprintf("copyleaks: %s %s: HTTP %s", e.Method, e.Endpoint, status)
	if body := strings.TrimSpace(string(e.Body)); body != "" {
		msg += ": " + truncate(body, maxErrorMessageBody)
	}
	return msg
}

// Is matches ErrUnauthorized (401), ErrRateLimited (429) and ErrUnderMaintenance (503).
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.StatusCode == http.StatusUnauthorized
	case ErrRateLimited:
		return e.StatusCode == http.StatusTooManyRequests
	case ErrUnderMaintenance:
		return e.StatusCode == http.StatusServiceUnavailable
	}
	return false
}

// invalid returns an error that matches ErrInvalidArgument.
func invalid(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidArgument, msg)
}

// truncate shortens s to at most n bytes without splitting a UTF-8 sequence.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "..."
}
