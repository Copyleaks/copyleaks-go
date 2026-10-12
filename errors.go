package copyleaks

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	// ErrUnauthorized matches an *APIError with HTTP status 401: the login token or the API
	// key was rejected. Check for it with errors.Is(err, ErrUnauthorized).
	ErrUnauthorized = errors.New("copyleaks: unauthorized")
	// ErrRateLimited matches an *APIError with HTTP status 429: too many requests were sent.
	// Wait before calling again.
	ErrRateLimited = errors.New("copyleaks: rate limited")
	// ErrUnderMaintenance matches an *APIError with HTTP status 503: the Copyleaks servers
	// are unavailable for maintenance. Retry later with exponential backoff.
	ErrUnderMaintenance = errors.New("copyleaks: under maintenance")
	// ErrTokenExpired is returned, before any request is sent, when the AuthToken passed to
	// a call has expired or expires within five minutes. Call Login again.
	ErrTokenExpired = errors.New("copyleaks: auth token expired")
	// ErrInvalidArgument is matched by the errors of the client-side checks, which run
	// before any request is sent: a missing required field, a malformed scan ID, a text
	// that is too short, a base URL that does not use https, and so on.
	ErrInvalidArgument = errors.New("copyleaks: invalid argument")
	// ErrResponseTooLarge is matched by the error of a call whose successful response body
	// is larger than the SDK accepts: 64 MiB for the usage history CSV and 16 MiB for
	// JSON. The body is not truncated; the call fails.
	ErrResponseTooLarge = errors.New("copyleaks: response too large")
)

// maxErrorMessageBody caps how much of the response body APIError.Error includes.
const maxErrorMessageBody = 512

// APIError is returned when the Copyleaks API answers with a non-2xx HTTP status.
//
// Use errors.Is with ErrUnauthorized, ErrRateLimited or ErrUnderMaintenance to check for
// those cases, and errors.As to read the status code and the response body.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// Method is the HTTP method of the request.
	Method string
	// Endpoint is the URL path of the request, such as "/v3/scans/submit/file/my-scan".
	Endpoint string
	// Body is the response body, up to 1 MiB. The server usually explains the error here.
	Body []byte
}

// Error returns the method, endpoint and status of the failed request, and the start of
// the response body.
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

// Is reports whether e matches target: ErrUnauthorized for status 401, ErrRateLimited for
// status 429 and ErrUnderMaintenance for status 503.
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
