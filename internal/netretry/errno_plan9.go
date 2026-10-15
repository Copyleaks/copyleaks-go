package netretry

// connErrnos is empty: Plan 9 reports connection errors as *net.OpError, which Transient accepts.
var connErrnos []error
