package genetlink

import (
	"encoding"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/mdlayher/netlink"
)

func TestConnExecute(t *testing.T) {
	req := Message{
		Header: Header{
			Command: 1,
			Version: 1,
		},
	}

	wantnl := netlink.Message{
		Header: netlink.Header{
			Type:  Protocol,
			Flags: netlink.HeaderFlagsRequest,
			// Sequence and PID not set because we are mocking the underlying
			// netlink connection.
		},
		Data: mustMarshal(req),
	}
	wantgenl := []Message{{
		Header: Header{
			Command: 1,
			Version: 1,
		},
		Data: []byte{0x01, 0x02, 0x03, 0x04},
	}}

	c, tc := testConn(t)
	tc.receive = []netlink.Message{{
		Header: netlink.Header{
			Length: 12,
			// Sequence and PID not set because we are mocking the underlying
			// netlink connection.
		},
		Data: []byte{
			0x01, 0x01, 0x00, 0x00,
			0x01, 0x02, 0x03, 0x04,
		},
	}}

	msgs, err := c.Execute(req, Controller, netlink.HeaderFlagsRequest)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if want, got := wantnl, tc.send; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected request:\n- want: %#v\n-  got: %#v",
			want, got)
	}
	if want, got := wantgenl, msgs; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected replies:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestConnSend(t *testing.T) {
	req := Message{
		Header: Header{
			Command: 1,
			Version: 1,
		},
	}

	c, tc := testConn(t)

	nlreq, err := c.Send(req, Controller, netlink.HeaderFlagsRequest)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	reqb, err := req.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	want := netlink.Message{
		Header: netlink.Header{
			Type:  Protocol,
			Flags: netlink.HeaderFlagsRequest,
		},
		Data: reqb,
	}

	if got := tc.send; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected output message from Conn.Send:\n- want: %#v\n-  got: %#v",
			want, got)
	}
	if got := nlreq; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected modified message:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestConnReceive(t *testing.T) {
	c, tc := testConn(t)
	tc.receive = []netlink.Message{
		{
			Header: netlink.Header{
				Length:   8,
				Sequence: 1,
				PID:      uint32(os.Getpid()),
			},
			Data: []byte{0x01, 0x01, 0x00, 0x00},
		},
		{
			Header: netlink.Header{
				Length:   12,
				Sequence: 1,
				PID:      uint32(os.Getpid()),
			},
			Data: []byte{
				0x02, 0x01, 0x00, 0x00,
				0x01, 0x02, 0x03, 0x04,
			},
		},
	}

	wantnl := tc.receive
	wantgenl := []Message{
		{
			Header: Header{
				Command: 1,
				Version: 1,
			},
			Data: make([]byte, 0),
		},
		{
			Header: Header{
				Command: 2,
				Version: 1,
			},
			Data: []byte{0x01, 0x02, 0x03, 0x04},
		},
	}

	genlmsgs, nlmsgs, err := c.Receive()
	if err != nil {
		t.Fatalf("failed to receive messages: %v", err)
	}

	if want, got := wantnl, nlmsgs; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected netlink.Messages from Conn.Receive:\n- want: %#v\n-  got: %#v",
			want, got)
	}

	if want, got := wantgenl, genlmsgs; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected Messages from Conn.Receive:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func testConn(t *testing.T) (*Conn, *testNetlinkConn) {
	c := &testNetlinkConn{}
	return newConn(c), c
}

type testNetlinkConn struct {
	send    netlink.Message
	receive []netlink.Message

	noopConn
}

func (c *testNetlinkConn) Send(m netlink.Message) (netlink.Message, error) {
	c.send = m
	return m, nil
}

func (c *testNetlinkConn) Receive() ([]netlink.Message, error) {
	return c.receive, nil
}

type noopConn struct{}

func (c *noopConn) Close() error                                    { return nil }
func (c *noopConn) Send(m netlink.Message) (netlink.Message, error) { return netlink.Message{}, nil }
func (c *noopConn) Receive() ([]netlink.Message, error)             { return nil, nil }

func mustMarshal(m encoding.BinaryMarshaler) []byte {
	b, err := m.MarshalBinary()
	if err != nil {
		panic(fmt.Sprintf("failed to marshal binary: %v", err))
	}

	return b
}

func mustMarshalAttributes(attrs []netlink.Attribute) []byte {
	b, err := netlink.MarshalAttributes(attrs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal attributes: %v", err))
	}

	return b
}
