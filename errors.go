package netlink

import (
	"errors"
	"fmt"
	"os"
)

// Error messages which can be returned by Validate.
var (
	errMismatchedSequence = errors.New("mismatched sequence in netlink reply")
	errMismatchedPID      = errors.New("mismatched PID in netlink reply")
	errShortErrorMessage  = errors.New("not enough data for netlink error code")
)

// Errors which can be returned by a Socket that does not implement
// all exposed methods of Conn.
var (
	errNotSupported = errors.New("operation not supported")
)

// notSupported provides a concise constructor for "not supported" errors.
func notSupported(op string) *OpError {
	return &OpError{
		Op:  op,
		Err: errNotSupported,
	}
}

// IsNotExist determines if an error is produced as the result of querying some
// file, object, resource, etc. which does not exist.  Users of this package
// should always use netlink.IsNotExist, rather than os.IsNotExist, when
// checking for specific netlink-related errors.
//
// Errors types created by this package, such as OpError, can be used with
// IsNotExist, but this function also defers to the behavior of os.IsNotExist
// for unrecognized error types.
func IsNotExist(err error) bool {
	switch err := err.(type) {
	case *OpError:
		// TODO(mdlayher): more error handling logic?

		// Unwrap the inner error and use the stdlib's logic.
		return os.IsNotExist(err.Err)
	default:
		return os.IsNotExist(err)
	}
}

var _ error = &OpError{}

// An OpError is an error produced as the result of a failed netlink operation.
type OpError struct {
	// Op is the operation which caused this OpError, such as "send"
	// or "receive".
	Op string

	// Err is the underlying error which caused this OpError.
	Err error
}

func (e *OpError) Error() string {
	if e == nil {
		return "<nil>"
	}

	return fmt.Sprintf("netlink %q: %v", e.Op, e.Err)
}
