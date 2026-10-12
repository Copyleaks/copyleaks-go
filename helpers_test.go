package copyleaks

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// lionsText is a 279-character sample, long enough for AI Text Detection.
const lionsText = "Lions are social animals, living in groups called prides, typically consisting of several females, " +
	"their offspring, and a few males. Female lions are the primary hunters, working together to catch prey. " +
	"Lions are known for their strength, teamwork, and complex social structures."

// testAccessToken is the access token of testToken.
const testAccessToken = "test-access-token-1234567890"

// testToken returns a token that stays valid for the whole test run.
func testToken() *AuthToken {
	return &AuthToken{AccessToken: testAccessToken, Issued: time.Now(), Expires: time.Now().Add(24 * time.Hour)}
}

// captured is one request received by a test server.
type captured struct {
	server string
	method string
	path   string
	query  url.Values
	header http.Header
	body   []byte
}

// testEnv runs a fake API server and a fake identity server. Both record every request
// and answer with the same status and body.
type testEnv struct {
	requests chan captured
	api      *httptest.Server
	identity *httptest.Server
}

// newTestEnv starts the servers of a testEnv and stops them when the test ends.
func newTestEnv(t *testing.T, status int, body string) *testEnv {
	t.Helper()
	env := &testEnv{requests: make(chan captured, 16)}
	handler := func(server string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			env.requests <- captured{
				server: server,
				method: r.Method,
				path:   r.URL.Path,
				query:  r.URL.Query(),
				header: r.Header.Clone(),
				body:   b,
			}
			w.WriteHeader(status)
			_, _ = io.WriteString(w, body)
		}
	}
	env.api = httptest.NewServer(handler("api"))
	t.Cleanup(env.api.Close)
	env.identity = httptest.NewServer(handler("identity"))
	t.Cleanup(env.identity.Close)
	return env
}

// client returns a Client that calls the test servers.
func (e *testEnv) client(t *testing.T, opts ...Option) *Client {
	t.Helper()
	all := []Option{WithAPIBaseURL(e.api.URL), WithIdentityBaseURL(e.identity.URL)}
	return mustClient(t, append(all, opts...)...)
}

// mustClient returns NewClient(opts...), failing the test on an error.
func mustClient(t *testing.T, opts ...Option) *Client {
	t.Helper()
	c, err := NewClient(opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// next returns the request that reached the servers, and fails the test when none did.
func (e *testEnv) next(t *testing.T) captured {
	t.Helper()
	select {
	case r := <-e.requests:
		return r
	default:
		t.Fatal("no request reached the server")
		return captured{}
	}
}

// none fails the test when a request reached the servers.
func (e *testEnv) none(t *testing.T) {
	t.Helper()
	select {
	case r := <-e.requests:
		t.Fatalf("unexpected %s %s request reached the server", r.method, r.path)
	default:
	}
}

// fixture returns the content of a file in testdata.
func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(b)
}

// assertJSONEqual fails the test when got and want are not the same JSON value.
func assertJSONEqual(t *testing.T, got []byte, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("body is not JSON: %v: %s", err, got)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("test JSON is invalid: %v", err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("JSON body\n got: %s\nwant: %s", got, want)
	}
}
