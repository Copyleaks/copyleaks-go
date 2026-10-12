package copyleaks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultAPIBaseURL is the base URL of the Copyleaks API.
	DefaultAPIBaseURL = "https://api.copyleaks.com"
	// DefaultIdentityBaseURL is the base URL of the Copyleaks identity server, used by Login.
	DefaultIdentityBaseURL = "https://id.copyleaks.com"
)

// defaultTimeout bounds each request when no http.Client is set with WithHTTPClient.
const defaultTimeout = 2 * time.Minute

// maxErrorBody caps how much of an error response body is kept in an APIError.
const maxErrorBody = 1 << 20

// maxDrain caps how much of an unread response body is discarded before the body is
// closed, so that the connection can be reused.
const maxDrain = 64 << 10

// maxCSVBody caps the size of a CSV response. It is a variable so that tests can lower it.
var maxCSVBody int64 = 64 << 20

// maxJSONBody caps the size of a JSON response. It is a variable so that tests can lower it.
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
	tenant          string
	userAgent       string
}

// Option configures a Client. Pass options to NewClient.
type Option func(*Client) error

// NewClient returns a Client for the production Copyleaks API, configured by opts. It
// returns an error that matches ErrInvalidArgument when an option is invalid, such as a
// base URL that does not use https.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		httpClient:      &http.Client{Timeout: defaultTimeout},
		apiBaseURL:      DefaultAPIBaseURL,
		identityBaseURL: DefaultIdentityBaseURL,
		userAgent:       sdkUserAgent,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// WithHTTPClient makes the Client send its requests with hc instead of a default
// http.Client that has a two-minute timeout. Use it to set timeouts or a proxy, or to
// add retries with a custom Transport; the SDK itself never retries. A nil hc is ignored.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc != nil {
			c.httpClient = hc
		}
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

// WithTenant sends tenant as the x-tenant header on every request of the Client,
// including Login and the public endpoints. WithCallTenant overrides it for one call.
// When no tenant is set, the header is not sent.
//
// The header only routes requests to an environment that Copyleaks directs you to. It does
// not grant access or separate data, so leave it unset unless Copyleaks tells you otherwise.
func WithTenant(tenant string) Option {
	return func(c *Client) error {
		c.tenant = tenant
		return nil
	}
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

// callTenantKey is the context key of the per-call tenant set by WithCallTenant.
type callTenantKey struct{}

// WithCallTenant returns a copy of ctx that makes a call send tenant as the x-tenant
// header, overriding the Client's WithTenant value for calls made with that context.
// An empty tenant omits the header, even when the Client has a tenant.
func WithCallTenant(ctx context.Context, tenant string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, callTenantKey{}, tenant)
}

// tenantFor returns the x-tenant value of a call: the per-call tenant when ctx has one,
// otherwise the Client's tenant. An empty result means the header is not sent.
func (c *Client) tenantFor(ctx context.Context) string {
	if tenant, ok := ctx.Value(callTenantKey{}).(string); ok {
		return tenant
	}
	return c.tenant
}

// apiRequest describes one HTTP call. path holds already escaped segments. identity
// selects the identity server instead of the API server. token is nil for public
// endpoints, and body, when not nil, is sent as JSON.
type apiRequest struct {
	method   string
	path     string
	query    url.Values
	identity bool
	token    *AuthToken
	body     any
}

// do sends r and decodes a successful response into out: the body is discarded when out
// is nil, copied when out is a *[]byte (up to 64 MiB), and decoded as JSON otherwise (up
// to 16 MiB). A larger body gives an error that matches ErrResponseTooLarge. A response
// with a non-2xx status becomes an *APIError.
func (c *Client) do(ctx context.Context, r apiRequest, out any) error {
	if c == nil {
		return invalid("nil *Client; create one with NewClient")
	}
	if ctx == nil {
		return invalid("nil context")
	}
	var body io.Reader
	if r.body != nil {
		b, err := json.Marshal(r.body)
		if err != nil {
			return fmt.Errorf("copyleaks: encode %s %s request: %w", r.method, r.path, err)
		}
		body = bytes.NewReader(b)
	}
	base := c.apiBaseURL
	if r.identity {
		base = c.identityBaseURL
	}
	target := base + r.path
	if len(r.query) > 0 {
		target += "?" + r.query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, r.method, target, body)
	if err != nil {
		return fmt.Errorf("copyleaks: build %s %s request: %w", r.method, r.path, err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	if r.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.token != nil {
		req.Header.Set("Authorization", "Bearer "+r.token.AccessToken)
	}
	if tenant := c.tenantFor(ctx); tenant != "" {
		req.Header.Set("x-tenant", tenant)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("copyleaks: %s %s: %w", r.method, r.path, err)
	}
	defer func() {
		// Discard what is left of the body, so that the connection can be reused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrain))
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return &APIError{StatusCode: resp.StatusCode, Method: r.method, Endpoint: r.path, Body: b}
	}
	switch v := out.(type) {
	case nil:
		return nil
	case *[]byte:
		b, err := readCapped(resp.Body, maxCSVBody)
		if err != nil {
			return fmt.Errorf("copyleaks: read %s %s response: %w", r.method, r.path, err)
		}
		*v = b
		return nil
	default:
		b, err := readCapped(resp.Body, maxJSONBody)
		if err != nil {
			return fmt.Errorf("copyleaks: read %s %s response: %w", r.method, r.path, err)
		}
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("copyleaks: decode %s %s response: %w", r.method, r.path, err)
		}
		return nil
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
