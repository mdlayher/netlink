package netlink

import (
	"errors"
	"math/rand"
	"sync/atomic"

	"golang.org/x/net/bpf"
)

// Error messages which can be returned by Validate.
var (
	errMismatchedSequence = errors.New("mismatched sequence in netlink reply")
	errMismatchedPID      = errors.New("mismatched PID in netlink reply")
	errShortErrorMessage  = errors.New("not enough data for netlink error code")
)

// Errors which can be returned by an osConn that does not implement
// all exposed methods of Conn.
var (
	errMulticastGroupsNotSupported = errors.New("multicast groups not supported")
	errBPFFiltersNotSupported      = errors.New("BPF filters not supported")
)

// A Conn is a connection to netlink.  A Conn can be used to send and
// receives messages to and from netlink.
type Conn struct {
	// osConn is the operating system-specific implementation of
	// a netlink sockets connection.
	c osConn

	// seq is an atomically incremented integer used to provide sequence
	// numbers when Conn.Send is called.
	seq *uint32

	// pid is the PID assigned by netlink.
	pid uint32
}

// An osConn is an operating-system specific implementation of netlink
// sockets used by Conn.
type osConn interface {
	Close() error
	Send(m Message) error
	Receive() ([]Message, error)
}

// Dial dials a connection to netlink, using the specified netlink family.
// Config specifies optional configuration for Conn.  If config is nil, a default
// configuration will be used.
func Dial(family int, config *Config) (*Conn, error) {
	// If configured, use the testing hook for osConn.
	if config != nil && config.Testing != nil {
		return newConn(config.Testing.(osConn), 1), nil
	}

	// Use OS-specific dial() to create osConn
	c, pid, err := dial(family, config)
	if err != nil {
		return nil, err
	}

	return newConn(c, pid), nil
}

// newConn is the internal constructor for Conn, used in tests.
func newConn(c osConn, pid uint32) *Conn {
	seq := rand.Uint32()

	return &Conn{
		c:   c,
		seq: &seq,
		pid: pid,
	}
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.c.Close()
}

// Execute sends a single Message to netlink using Conn.Send, receives one or more
// replies using Conn.Receive, and then checks the validity of the replies against
// the request using Validate.
//
// See the documentation of Conn.Send, Conn.Receive, and Validate for details about
// each function.
func (c *Conn) Execute(m Message) ([]Message, error) {
	req, err := c.Send(m)
	if err != nil {
		return nil, err
	}

	replies, err := c.Receive()
	if err != nil {
		return nil, err
	}

	if err := Validate(req, replies); err != nil {
		return nil, err
	}

	return replies, nil
}

// Send sends a single Message to netlink.  In most cases, m.Header's Length,
// Sequence, and PID fields should be set to 0, so they can be populated
// automatically before the Message is sent.  On success, Send returns a copy
// of the Message with all parameters populated, for later validation.
//
// If m.Header.Length is 0, it will be automatically populated using the
// correct length for the Message, including its payload.
//
// If m.Header.Sequence is 0, it will be automatically populated using the
// next sequence number for this connection.
//
// If m.Header.PID is 0, it will be automatically populated using a PID
// assigned by netlink.
func (c *Conn) Send(m Message) (Message, error) {
	ml := nlmsgLength(len(m.Data))

	// TODO(mdlayher): fine-tune this limit.
	if ml > (1024 * 32) {
		return Message{}, errors.New("netlink message data too large")
	}

	if m.Header.Length == 0 {
		m.Header.Length = uint32(nlmsgAlign(ml))
	}

	if m.Header.Sequence == 0 {
		m.Header.Sequence = c.nextSequence()
	}

	if m.Header.PID == 0 {
		m.Header.PID = c.pid
	}

	if err := c.c.Send(m); err != nil {
		return Message{}, err
	}

	return m, nil
}

// Receive receives one or more messages from netlink.  Multi-part messages are
// handled transparently and returned as a single slice of Messages, with the
// final empty "multi-part done" message removed.
//
// If any of the messages indicate a netlink error, that error will be returned.
func (c *Conn) Receive() ([]Message, error) {
	msgs, err := c.receive()
	if err != nil {
		return nil, err
	}

	// Trim the final message with multi-part done indicator if
	// present
	if m := msgs[len(msgs)-1]; m.Header.Flags&HeaderFlagsMulti != 0 && m.Header.Type == HeaderTypeDone {
		return msgs[:len(msgs)-1], nil
	}

	return msgs, nil
}

// receive is the internal implementation of Conn.Receive, which can be called
// recursively to handle multi-part messages.
func (c *Conn) receive() ([]Message, error) {
	msgs, err := c.c.Receive()
	if err != nil {
		return nil, err
	}

	// If this message is multi-part, we will need to perform an recursive call
	// to continue draining the socket
	var multi bool

	for _, m := range msgs {
		// Is this a multi-part message and is it not done yet?
		if m.Header.Flags&HeaderFlagsMulti != 0 && m.Header.Type != HeaderTypeDone {
			multi = true
		}

		if err := checkMessage(m); err != nil {
			return nil, err
		}
	}

	if !multi {
		return msgs, nil
	}

	// More messages waiting
	mmsgs, err := c.receive()
	if err != nil {
		return nil, err
	}

	return append(msgs, mmsgs...), nil
}

// A groupJoinLeaver is an osConn that supports joining and leaving
// netlink multicast groups.
type groupJoinLeaver interface {
	osConn
	JoinGroup(group uint32) error
	LeaveGroup(group uint32) error
}

// JoinGroup joins a netlink multicast group by its ID.
func (c *Conn) JoinGroup(group uint32) error {
	gc, ok := c.c.(groupJoinLeaver)
	if !ok {
		return errMulticastGroupsNotSupported
	}

	return gc.JoinGroup(group)
}

// LeaveGroup leaves a netlink multicast group by its ID.
func (c *Conn) LeaveGroup(group uint32) error {
	gc, ok := c.c.(groupJoinLeaver)
	if !ok {
		return errMulticastGroupsNotSupported
	}

	return gc.LeaveGroup(group)
}

// A bpfSetter is an osConn that supports setting BPF filters.
type bpfSetter interface {
	osConn
	SetBPF(filter []bpf.RawInstruction) error
}

// SetBPF attaches an assembled BPF program to a Conn.
func (c *Conn) SetBPF(filter []bpf.RawInstruction) error {
	bc, ok := c.c.(bpfSetter)
	if !ok {
		return errBPFFiltersNotSupported
	}

	return bc.SetBPF(filter)
}

// nextSequence atomically increments Conn's sequence number and returns
// the incremented value.
func (c *Conn) nextSequence() uint32 {
	return atomic.AddUint32(c.seq, 1)
}

// Validate validates one or more reply Messages against a request Message,
// ensuring that they contain matching sequence numbers and PIDs.
func Validate(request Message, replies []Message) error {
	for _, m := range replies {
		// Check for mismatched sequence, unless:
		//   - request had no sequence, meaning we are probably validating
		//     a multicast reply
		if m.Header.Sequence != request.Header.Sequence && request.Header.Sequence != 0 {
			return errMismatchedSequence
		}

		// Check for mismatched PID, unless:
		//   - request had no PID, meaning we are either:
		//     - validating a multicast reply
		//     - netlink has not yet assigned us a PID
		//   - response had no PID, meaning it's from the kernel as a multicast reply
		if m.Header.PID != request.Header.PID && request.Header.PID != 0 && m.Header.PID != 0 {
			return errMismatchedPID
		}
	}

	return nil
}

// Config contains options for a Conn.
type Config struct {
	// Groups is a bitmask which specifies multicast groups. If set to 0,
	// no multicast group subscriptions will be made.
	Groups uint32

	// Testing is a special hook used to configure a netlink.Conn for
	// testing with nltest.  It should not be used by other applications.
	Testing interface{}
}
