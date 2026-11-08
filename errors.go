package copyleaks

import (
	"encoding/json"
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

// The error types of APIErrorBody.Type and WebhookError.Type, from
// https://docs.copyleaks.com/using-the-apis/api-errors/. Type stays a plain string, so a type that the API adds later
// still comes through.
const (
	// ErrorTypeInvalidRequest is the type of an error caused by the request, such as a missing parameter.
	ErrorTypeInvalidRequest = "invalid_request_error"
	// ErrorTypeAuthentication is the type of an authentication error.
	ErrorTypeAuthentication = "authentication_error"
	// ErrorTypePayment is the type of a payment error.
	ErrorTypePayment = "payment_error"
	// ErrorTypeAPI is the type of an error on the Copyleaks side.
	ErrorTypeAPI = "api_error"
)

// APIErrorBody is the error object that the API sends in the body of an error response:
// {"error": {"type": ..., "id": ..., "code": ..., "message": ..., "url": ..., "details": [...]}}.
// See https://docs.copyleaks.com/using-the-apis/api-errors/. Read it with APIError.ErrorBody.
// A field that the server did not send is empty. It is the same object as WebhookError, which an error webhook
// carries.
type APIErrorBody struct {
	// Type is the error type: ErrorTypeInvalidRequest, ErrorTypeAuthentication, ErrorTypePayment, ErrorTypeAPI, or
	// a type that the API adds later.
	Type string
	// ID is the machine-readable error ID, such as "missing_parameter".
	ID string
	// Code is the numeric error code, or 0 when the server sent none.
	Code int
	// Message is a human-readable description of the error.
	Message string
	// URL links to the documentation of the error.
	URL string
	// Details describe the request parameters that caused the error; nil when the server sent none.
	Details []APIErrorDetail
}

// APIErrorDetail describes one request parameter that caused an API error.
type APIErrorDetail struct {
	// Param is the name of the parameter.
	Param string
	// Message describes the problem with the parameter.
	Message string
}

// ErrorBody parses Body as the error object of the API (see APIErrorBody). It never fails: when Body is not a JSON
// object with an "error" object, every field is empty, and a field of an unexpected JSON type is empty too. Code
// accepts a JSON integer or a numeric string, with surrounding whitespace; Details keeps the entries that are JSON
// objects.
func (e *APIError) ErrorBody() APIErrorBody {
	var out APIErrorBody
	var envelope, obj map[string]json.RawMessage
	if json.Unmarshal(e.Body, &envelope) != nil || json.Unmarshal(envelope["error"], &obj) != nil || obj == nil {
		return out
	}
	out.Type = jsonString(obj["type"])
	out.ID = jsonString(obj["id"])
	out.Code = jsonCode(obj["code"])
	out.Message = jsonString(obj["message"])
	out.URL = jsonString(obj["url"])
	var details []json.RawMessage
	if json.Unmarshal(obj["details"], &details) == nil {
		for _, d := range details {
			var detail map[string]json.RawMessage
			if json.Unmarshal(d, &detail) != nil || detail == nil {
				continue
			}
			out.Details = append(out.Details, APIErrorDetail{Param: jsonString(detail["param"]), Message: jsonString(detail["message"])})
		}
	}
	return out
}

// jsonString returns the JSON string raw, or "" when raw is missing or not a string.
func jsonString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return s
}

// jsonCode returns the JSON integer or numeric string raw (surrounding whitespace ignored), or 0 when raw is missing
// or anything else.
func jsonCode(raw json.RawMessage) int {
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return n
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return n
		}
	}
	return 0
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
