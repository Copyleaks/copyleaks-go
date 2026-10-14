package copyleaks

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDefaultTransport(t *testing.T) {
	before := http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost
	c := mustClient(t)
	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("default transport is %T, want *http.Transport", c.httpClient.Transport)
	}
	if tr == http.DefaultTransport {
		t.Error("the default client shares http.DefaultTransport")
	}
	if tr.MaxIdleConnsPerHost != 32 || !tr.ForceAttemptHTTP2 || tr.Proxy == nil || tr.TLSHandshakeTimeout == 0 {
		t.Errorf("transport: MaxIdleConnsPerHost %d, ForceAttemptHTTP2 %v", tr.MaxIdleConnsPerHost, tr.ForceAttemptHTTP2)
	}
	if after := http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost; after != before {
		t.Errorf("NewClient changed http.DefaultTransport.MaxIdleConnsPerHost from %d to %d", before, after)
	}
	if c.httpClient.Timeout != 2*time.Minute {
		t.Errorf("default timeout = %v, want 2m", c.httpClient.Timeout)
	}
	if c2 := mustClient(t); c2.httpClient.Transport == c.httpClient.Transport {
		t.Error("two clients share one transport")
	}
}

func TestUserHTTPClientIsNotChanged(t *testing.T) {
	transport := &countingTransport{}
	hc := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	env := newTestEnv(t, http.StatusOK, `["en"]`)
	c := env.client(t, WithHTTPClient(hc), WithTimeout(time.Hour), WithRetry(RetryPolicy{MaxAttempts: 3}))
	if c.httpClient != hc {
		t.Fatal("the Client does not use the given http.Client")
	}
	if _, err := c.OCRSupportedLanguages(context.Background()); err != nil {
		t.Fatal(err)
	}
	if hc.Transport != transport || hc.Timeout != 5*time.Second || hc.CheckRedirect != nil || hc.Jar != nil {
		t.Errorf("the given http.Client was changed: %+v", hc)
	}
	if transport.count != 1 {
		t.Errorf("%d requests through the given transport, want 1", transport.count)
	}
}

func TestWithTimeout(t *testing.T) {
	if c := mustClient(t, WithTimeout(10*time.Minute)); c.httpClient.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want 10m", c.httpClient.Timeout)
	}
	if c := mustClient(t, WithTimeout(0)); c.httpClient.Timeout != 0 {
		t.Errorf("Timeout = %v, want none", c.httpClient.Timeout)
	}
	if _, err := NewClient(WithTimeout(-time.Second)); !errors.Is(err, ErrInvalidArgument) || !strings.Contains(err.Error(), "WithTimeout") {
		t.Errorf("negative timeout: err = %v", err)
	}
}

func TestWithMaxJSONResponseSize(t *testing.T) {
	env := newTestEnv(t, http.StatusOK, `{"Amount":123456}`)
	if _, err := env.client(t, WithMaxJSONResponseSize(16)).CreditsBalance(context.Background(), testToken()); !errors.Is(err, ErrResponseTooLarge) {
		t.Errorf("err = %v, want ErrResponseTooLarge", err)
	}
	env = newTestEnv(t, http.StatusOK, `{"Amount":123456}`)
	if res, err := env.client(t, WithMaxJSONResponseSize(17)).CreditsBalance(context.Background(), testToken()); err != nil || res.Amount != 123456 {
		t.Errorf("at the limit: %+v, %v", res, err)
	}
	for _, size := range []int64{0, -1} {
		if _, err := NewClient(WithMaxJSONResponseSize(size)); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("size %d: err = %v", size, err)
		}
	}
}
