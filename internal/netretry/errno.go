//go:build !plan9

package netretry

import "syscall"

// connErrnos are the system errors of a refused, reset or broken connection.
var connErrnos = []error{syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ECONNABORTED, syscall.EPIPE}
