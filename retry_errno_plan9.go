package copyleaks

// connErrnos are the system errors of a refused, reset or broken connection. Plan 9
// reports them as *net.OpError, which transientNetError also accepts.
var connErrnos []error
