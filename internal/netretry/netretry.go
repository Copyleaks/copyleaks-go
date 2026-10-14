// Package netretry tells the connection errors that a retry may fix, such as a reset
// connection, from the ones that it cannot, such as an invalid TLS certificate.
package netretry

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"strings"
)

// Transient reports whether err, an error of sending a request or of reading its
// response, is a connection error that a retry may fix: a timeout, a refused, reset or
// dropped connection, or an HTTP/2 connection that the server shut down, such as with
// GOAWAY. TLS certificate errors and unknown hosts are not transient, as a retry fails
// the same way.
func Transient(err error) bool {
	var certErr *tls.CertificateVerificationError
	var authorityErr x509.UnknownAuthorityError
	var hostnameErr x509.HostnameError
	var invalidErr x509.CertificateInvalidError
	if errors.As(err, &certErr) || errors.As(err, &authorityErr) || errors.As(err, &hostnameErr) || errors.As(err, &invalidErr) {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	for _, errno := range connErrnos {
		if errors.Is(err, errno) {
			return true
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// A TLS alert from the server (for example a rejected handshake) is not a connection problem.
		return opErr.Op != "remote error"
	}
	msg := err.Error()
	for _, m := range http2ConnErrors {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}

// http2ConnErrors are the messages of the errors of the HTTP/2 transport that report a
// connection that the server shut down or lost, or a request that it refused before
// processing it. The transport does not export their types.
var http2ConnErrors = []string{
	"http2: server sent GOAWAY",
	"http2: Transport received Server's graceful shutdown GOAWAY",
	"http2: client connection lost",
	"http2: client conn not usable",
	"http2: client conn is closed",
	"REFUSED_STREAM",
}
