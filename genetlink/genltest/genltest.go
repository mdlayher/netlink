// Package genltest provides utilities for generic netlink testing.
package genltest

import (
	"fmt"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"
	"github.com/mdlayher/netlink/nltest"
)

// Error returns a netlink error to the caller with the specified error
// number.
func Error(number int) error {
	return &errnoError{number: number}
}

type errnoError struct {
	number int
}

func (err *errnoError) Error() string {
	return fmt.Sprintf("genltest errno: %d", err.number)
}

// A Func is a function that can be used to test genetlink.Conn interactions.
// The function can choose to return zero or more generic netlink messages,
// or an error if needed.
//
// For a netlink request/response interaction, the requests greq and nreq are
// populated by genetlink.Conn.Send and passed to the function.  greq is created
// from the body of nreq.
//
// For multicast interactions, both greq and nreq are empty when passed to the function
// when genetlink.Conn.Receive is called.
type Func func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error)

// Dial sets up a genetlink.Conn for testing using the specified Func. All requests
// sent from the connection will be passed to the Func.  The connection should be
// closed as usual when it is no longer needed.
func Dial(fn Func) *genetlink.Conn {
	const pid = 1

	sock := nltest.NewSocket(adapt(fn))

	return genetlink.NewConn(netlink.NewConn(sock, pid))
}

var _ nltest.Func = adapt(nil)

// adapt is an adapter function for a Func to be used as a nltest.Func.  adapt
// handles marshaling and unmarshaling of generic netlink messages.
func adapt(fn Func) nltest.Func {
	return func(req netlink.Message) ([]netlink.Message, error) {
		var gm genetlink.Message

		// Populate message if some data has been passed in req.
		if len(req.Data) > 0 {
			if err := gm.UnmarshalBinary(req.Data); err != nil {
				return nil, err
			}
		}

		gmsgs, err := fn(gm, req)
		if err != nil {
			// An error was returned with an error number by the Func.
			// Pass this to the caller as a netlink message error.
			nerr, ok := err.(*errnoError)
			if !ok {
				return nil, err
			}

			return nltest.Error(nerr.number, req)
		}

		nmsgs := make([]netlink.Message, 0, len(gmsgs))
		for _, msg := range gmsgs {
			b, err := msg.MarshalBinary()
			if err != nil {
				return nil, err
			}

			nmsgs = append(nmsgs, netlink.Message{
				// Mimic the sequence and PID of the request for validation.
				Header: netlink.Header{
					Sequence: req.Header.Sequence,
					PID:      req.Header.PID,
				},
				Data: b,
			})
		}

		return nmsgs, nil
	}
}
