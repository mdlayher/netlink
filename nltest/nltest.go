// Package nltest provides utilities for netlink testing.
package nltest

import (
	"fmt"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
)

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
type Func func(req netlink.Message) ([]netlink.Message, error)

// Dial sets up a netlink.Conn for testing using the specified Func. All requests
// sent from the connection will be passed to the Func.  The connection should be
// closed as usual when it is no longer needed.
func Dial(fn Func) *netlink.Conn {
	cfg := &netlink.Config{
		// TODO(mdlayher): consider exposing a proper API in netlink for allowing
		// arbitrary netlink.osConn implementations over any transport, removing
		// the need for this "hack".
		Testing: &conn{fn: fn},
	}

	c, err := netlink.Dial(0, cfg)
	if err != nil {
		panic(fmt.Sprintf("nltest setup error: %v", err))
	}

	return c
}

// A conn is a netlink.osConn used for testing.  Its methods must match those of
// netlink.osConn or Dial will panic.
type conn struct {
	fn Func

	msgs []netlink.Message
	err  error
}

func (c *conn) Close() error { return nil }

func (c *conn) Send(m netlink.Message) error {
	c.msgs, c.err = c.fn(m)
	return nil
}

func (c *conn) Receive() ([]netlink.Message, error) {
	// No messages set by Send means that we are emulating a
	// multicast response.
	if len(c.msgs) == 0 {
		return c.fn(netlink.Message{})
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
