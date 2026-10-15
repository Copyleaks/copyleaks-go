package copyleaks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// tokenExpiryMargin treats a token as expired this early, so it does not expire mid-request.
const tokenExpiryMargin = 5 * time.Minute

// AuthToken is the login token returned by Login. It is valid for 48 hours; reuse it until it expires.
// Treat it like a password: fmt prints it redacted, but its JSON (for storing it) holds the access token.
type AuthToken struct {
	// AccessToken is the bearer token sent in the Authorization header.
	AccessToken string `json:"access_token"`
	// Issued is when the token was issued.
	Issued time.Time `json:".issued"`
	// Expires is when the token expires.
	Expires time.Time `json:".expires"`
}

// UnmarshalJSON decodes a login response or a stored token; times without a UTC offset are UTC.
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

// Format implements fmt.Formatter: every verb and flag prints the redacted String form.
func (t AuthToken) Format(f fmt.State, verb rune) {
	_, _ = io.WriteString(f, t.String())
}

// parseTokenTime parses RFC 3339, or a time without a UTC offset as UTC; "" gives the zero time.
func parseTokenTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05.999999999", s)
}

// VerifyAuthToken checks a token locally: ErrInvalidArgument when it is nil or empty, ErrTokenExpired when it
// expired or expires within five minutes. Every authenticated method runs it first.
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

// Login exchanges an account email and API key for an AuthToken. Only 12 logins per account are allowed in
// 15 minutes, so reuse the token. See https://docs.copyleaks.com/reference/actions/account/login.
func (c *Client) Login(ctx context.Context, email, key string) (*AuthToken, error) {
	if email == "" || key == "" {
		return nil, invalid("email and key are required")
	}
	var token AuthToken
	err := c.do(ctx, apiRequest{
		method:     http.MethodPost,
		path:       "/v3/account/login/api",
		identity:   true,
		body:       loginRequest{Email: email, Key: key},
		idempotent: true,
	}, &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}
