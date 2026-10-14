//go:build !plan9

package netretry

import (
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
)

// timeoutError is a net.Error that reports a timeout.
type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestTransient(t *testing.T) {
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
		{"tls alert", wrap(&net.OpError{Op: "remote error", Err: errors.New("tls: handshake failure")}), false},
		{"redirects", wrap(errors.New("stopped after 10 redirects")), false},
		{"other", errors.New("something else"), false},
	}
	for _, tt := range tests {
		if got := Transient(tt.err); got != tt.want {
			t.Errorf("%s: Transient(%v) = %v, want %v", tt.name, tt.err, got, tt.want)
		}
	}
}
