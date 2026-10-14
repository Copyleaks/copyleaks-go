package netretry

// connErrnos are the system errors of a refused, reset or broken connection. Plan 9
// reports them as *net.OpError, which Transient also accepts.
var connErrnos []error
