//go:build !plan9

package copyleaks

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestRetryConnectionRefused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	c, slept := retryClientFor(t, "http://"+addr, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
	var opErr *net.OpError
	if _, err := c.CreditsBalance(context.Background(), testToken()); !errors.As(err, &opErr) || opErr.Op != "dial" {
		t.Fatalf("err = %v, want a refused connection", err)
	}
	if len(*slept) != 2 {
		t.Errorf("%d waits, want 2", len(*slept))
	}
}

// timeoutError is a net.Error that reports a timeout.
type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestTransientNetError(t *testing.T) {
	wrap := func(err error) error { return &url.Error{Op: "Post", URL: "https://api.copyleaks.com/v3", Err: err} }
	dial := func(err error) error { return wrap(&net.OpError{Op: "dial", Net: "tcp", Err: err}) }
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"refused", dial(os.NewSyscallError("connect", syscall.ECONNREFUSED)), true},
		{"reset", wrap(&net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("read", syscall.ECONNRESET)}), true},
		{"bare reset", syscall.ECONNRESET, true},
		{"broken pipe", fmt.Errorf("write: %w", syscall.EPIPE), true},
		{"dropped", wrap(io.EOF), true},
		{"truncated body", io.ErrUnexpectedEOF, true},
		{"timeout", wrap(timeoutError{}), true},
		{"other dial error", dial(errors.New("network is unreachable")), true},
		{"dns timeout", dial(&net.DNSError{Name: "api.copyleaks.com", Err: "i/o timeout", IsTimeout: true}), true},
		{"goaway", wrap(errors.New("http2: server sent GOAWAY and closed the connection; LastStreamID=1, ErrCode=NO_ERROR, debug=\"\"")), true},
		{"graceful goaway", wrap(errors.New("http2: Transport received Server's graceful shutdown GOAWAY")), true},
		{"refused stream", wrap(errors.New("stream error: stream ID 3; REFUSED_STREAM")), true},
		{"unknown host", dial(&net.DNSError{Name: "api.example.invalid", Err: "no such host", IsNotFound: true}), false},
		{"untrusted certificate", wrap(&tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}), false},
		{"untrusted certificate of a dial", dial(&tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}), false},
		{"wrong host certificate", dial(x509.HostnameError{Certificate: &x509.Certificate{}, Host: "api.copyleaks.com"}), false},
		{"expired certificate", dial(x509.CertificateInvalidError{Reason: x509.Expired}), false},
		{"unknown authority", dial(x509.UnknownAuthorityError{}), false},
		{"redirects", wrap(errors.New("stopped after 10 redirects")), false},
		{"other", errors.New("something else"), false},
	}
	for _, tt := range tests {
		if got := transientNetError(tt.err); got != tt.want {
			t.Errorf("%s: transientNetError(%v) = %v, want %v", tt.name, tt.err, got, tt.want)
		}
	}
}
