package genetlink

import "github.com/mdlayher/netlink"

// Family is the netlink family constant used to specify generic netlink.
const Family = 16

// A Conn is a generic netlink connection.  A Conn can be used to send and
// receive generic netlink messages to and from netlink.
type Conn struct {
	c conn
}

var _ conn = &netlink.Conn{}

// A conn is a netlink connection, which can be swapped for tests.
type conn interface {
	Close() error
	Send(m netlink.Message) (netlink.Message, error)
	Receive() ([]netlink.Message, error)
}

// Dial dials a generic netlink connection.  Config specifies optional
// configuration for the underlying netlink connection.  If config is
// nil, a default configuration will be used.
func Dial(config *netlink.Config) (*Conn, error) {
	c, err := netlink.Dial(Family, config)
	if err != nil {
		return nil, err
	}

	return newConn(c), nil
}

// newConn is the internal constructor for Conn, used in tests.
func newConn(c conn) *Conn {
	return &Conn{
		c: c,
	}
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.c.Close()
}

// Send sends a single Message to netlink, wrapping it in a netlink.Message
// using the specified flags.  On success, Send returns a copy of the
// netlink.Message with all parameters populated, for later validation.
func (c *Conn) Send(m Message, flags netlink.HeaderFlags) (netlink.Message, error) {
	nm := netlink.Message{
		Header: netlink.Header{
			Type:  Family,
			Flags: flags,
		},
	}

	mb, err := m.MarshalBinary()
	if err != nil {
		return netlink.Message{}, err
	}
	nm.Data = mb

	reqnm, err := c.c.Send(nm)
	if err != nil {
		return netlink.Message{}, err
	}

	return reqnm, nil
}

// Receive receives one or more Messages from netlink.  The netlink.Messages
// used to wrap each Message are available for later validation.
func (c *Conn) Receive() ([]Message, []netlink.Message, error) {
	msgs, err := c.c.Receive()
	if err != nil {
		return nil, nil, err
	}

	gmsgs := make([]Message, 0, len(msgs))
	for _, nm := range msgs {
		var gm Message
		if err := (&gm).UnmarshalBinary(nm.Data); err != nil {
			return nil, nil, err
		}

		gmsgs = append(gmsgs, gm)
	}

	return gmsgs, msgs, nil
}

// Execute sends a single Message to netlink using Conn.Send, receives one or
// more replies using Conn.Receive, and then checks the validity of the replies
// against the request using netlink.Validate.
//
// See the documentation of Conn.Send, Conn.Receive, and netlink.Validate for
// details about each function.
func (c *Conn) Execute(m Message, flags netlink.HeaderFlags) ([]Message, error) {
	req, err := c.Send(m, flags)
	if err != nil {
		return nil, err
	}

	msgs, replies, err := c.Receive()
	if err != nil {
		return nil, err
	}

	if err := netlink.Validate(req, replies); err != nil {
		return nil, err
	}

	return msgs, nil
}
