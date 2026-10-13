package copyleaks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// tokenExpiryMargin is how long before its expiry a token is treated as expired, so that
// it does not expire while a request is in flight.
const tokenExpiryMargin = 5 * time.Minute

// AuthToken is the login token returned by Login. Pass it to every authenticated call.
//
// A token is valid for 48 hours and can be reused for many calls; check it with
// VerifyAuthToken and log in again when it expires. Treat it like a password. It can be
// stored with encoding/json and read back; the JSON holds the access token. Printing it
// with the fmt package, with any verb, redacts the access token.
type AuthToken struct {
	// AccessToken is the bearer token sent in the Authorization header.
	AccessToken string `json:"access_token"`
	// Issued is when the token was issued.
	Issued time.Time `json:".issued"`
	// Expires is when the token expires.
	Expires time.Time `json:".expires"`
}

// UnmarshalJSON decodes a login response or a stored token. Times without a UTC offset
// are read as UTC.
func (t *AuthToken) UnmarshalJSON(b []byte) error {
	var raw struct {
		AccessToken string `json:"access_token"`
		Issued      string `json:".issued"`
		Expires     string `json:".expires"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	issued, err := parseTokenTime(raw.Issued)
	if err != nil {
		return fmt.Errorf("copyleaks: auth token .issued: %w", err)
	}
	expires, err := parseTokenTime(raw.Expires)
	if err != nil {
		return fmt.Errorf("copyleaks: auth token .expires: %w", err)
	}
	*t = AuthToken{AccessToken: raw.AccessToken, Issued: issued, Expires: expires}
	return nil
}

// String returns the token with the access token redacted.
func (t AuthToken) String() string {
	return fmt.Sprintf("AuthToken{AccessToken: [REDACTED], Issued: %s, Expires: %s}",
		t.Issued.Format(time.RFC3339), t.Expires.Format(time.RFC3339))
}

// Format implements fmt.Formatter, so that every fmt verb and flag, such as %v, %+v,
// %#v, %s, %q, %x and %d, prints the redacted String form and never the access token.
func (t AuthToken) Format(f fmt.State, verb rune) {
	_, _ = io.WriteString(f, t.String())
}

// parseTokenTime parses an RFC 3339 time, or one without a UTC offset as UTC.
// An empty string gives the zero time.
func parseTokenTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05.999999999", s)
}

// VerifyAuthToken checks a token without calling the API. It returns nil when the token
// has an access token and does not expire within the next five minutes. It returns an
// error that matches ErrTokenExpired when the token expired or is about to, and one that
// matches ErrInvalidArgument when the token is nil or empty.
//
// Every authenticated method runs this check before sending its request.
func VerifyAuthToken(token *AuthToken) error {
	if token == nil || token.AccessToken == "" {
		return invalid("auth token is missing; call Login first")
	}
	if !token.Expires.After(time.Now().Add(tokenExpiryMargin)) {
		return fmt.Errorf("%w: it expired or expires within %s; call Login again", ErrTokenExpired, tokenExpiryMargin)
	}
	return nil
}

// loginRequest is the body of Login.
type loginRequest struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

// Login exchanges the email and the API key of a Copyleaks account for an AuthToken.
// The identity server allows 12 logins per account in 15 minutes, so reuse the token
// until it expires.
//
// See https://docs.copyleaks.com/reference/actions/account/login.
func (c *Client) Login(ctx context.Context, email, key string) (*AuthToken, error) {
	if email == "" || key == "" {
		return nil, invalid("email and key are required")
	}
	var token AuthToken
	err := c.do(ctx, apiRequest{
		method:   http.MethodPost,
		path:     "/v3/account/login/api",
		identity: true,
		body:     loginRequest{Email: email, Key: key},
	}, &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}
