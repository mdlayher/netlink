// Package nltest provides utilities for netlink testing.
package nltest

import (
	"io"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
)

// PID is the netlink header PID value assigned by nltest.
const PID = 1

// Multipart sends a slice of netlink.Messages to the caller as a
// netlink multi-part message. If less than two messages are present,
// the messages are not altered.
func Multipart(msgs []netlink.Message) ([]netlink.Message, error) {
	if len(msgs) < 2 {
		return msgs, nil
	}

	for i := range msgs {
		// Last message has header type "done" in addition to multi-part flag.
		if i == len(msgs)-1 {
			msgs[i].Header.Type = netlink.HeaderTypeDone
		}

		msgs[i].Header.Flags |= netlink.HeaderFlagsMulti
	}

	return msgs, nil
}

// Error returns a netlink error to the caller with the specified error
// number, in the body of the specified request message.
func Error(number int, req netlink.Message) ([]netlink.Message, error) {
	req.Header.Length += 4
	req.Header.Type = netlink.HeaderTypeError

	errno := -1 * int32(number)
	req.Data = append(nlenc.Int32Bytes(errno), req.Data...)

	return []netlink.Message{req}, nil
}

// A Func is a function that can be used to test netlink.Conn interactions.
// The function can choose to return zero or more netlink messages, or an
// error if needed.
//
// For a netlink request/response interaction, a request req is populated by
// netlink.Conn.Send and passed to the function.
//
// For multicast interactions, an empty request req is passed to the function
// when netlink.Conn.Receive is called.
//
// If a Func returns an error, the error will be returned as-is to the caller.
// If no messages and io.EOF are returned, no messages and no error will be
// returned to the caller, simulating a multi-part message with no data.
type Func func(req netlink.Message) ([]netlink.Message, error)

// Dial sets up a netlink.Conn for testing using the specified Func. All requests
// sent from the connection will be passed to the Func.  The connection should be
// closed as usual when it is no longer needed.
func Dial(fn Func) *netlink.Conn {
	return netlink.NewConn(NewSocket(fn), PID)
}

// NewSocket creates a netlink.Socket which passes requests to Func.  NewSocket
// is primarily useful for building higher-level netlink testing packages on
// top of nltest.
func NewSocket(fn Func) netlink.Socket {
	return &socket{
		fn: fn,
	}
}

var _ netlink.Socket = &socket{}

// A socket is a netlink.Socket used for testing.
type socket struct {
	fn Func

	msgs []netlink.Message
	err  error
}

func (c *socket) Close() error { return nil }

func (c *socket) Send(m netlink.Message) error {
	c.msgs, c.err = c.fn(m)
	return nil
}

func (c *socket) Receive() ([]netlink.Message, error) {
	// No messages set by Send means that we are emulating a
	// multicast response or an error occurred.
	if len(c.msgs) == 0 {
		switch c.err {
		case nil:
			// No error, simulate multicast, but also return EOF to simulate
			// no replies if needed.
			msgs, err := c.fn(netlink.Message{})
			if err == io.EOF {
				err = nil
			}

			return msgs, err
		case io.EOF:
			// EOF, simulate no replies in multi-part message.
			return nil, nil
		default:
			// Some error occurred and should be passed to the caller.
			return nil, c.err
		}
	}

	// Detect multi-part messages.
	var multi bool
	for _, m := range c.msgs {
		if m.Header.Flags&netlink.HeaderFlagsMulti != 0 && m.Header.Type != netlink.HeaderTypeDone {
			multi = true
		}
	}

	// When a multi-part message is detected, return all messages except for the
	// final "multi-part done", so that a second call to Receive from netlink.Conn
	// will drain that message.
	if multi {
		last := c.msgs[len(c.msgs)-1]
		ret := c.msgs[:len(c.msgs)-1]
		c.msgs = []netlink.Message{last}

		return ret, c.err
	}

	msgs, err := c.msgs, c.err
	c.msgs, c.err = nil, nil

	return msgs, err
}
