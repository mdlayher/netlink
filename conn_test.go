package netlink

import (
	"os"
	"reflect"
	"testing"
)

func TestConnExecute(t *testing.T) {
	req := Message{
		Header: Header{
			Flags:    HeaderFlagsRequest | HeaderFlagsAcknowledge,
			Sequence: 1,
		},
	}

	pid := uint32(os.Getpid())

	replies := []Message{{
		Header: Header{
			Type:     HeaderTypeError,
			Sequence: 1,
			PID:      pid,
		},
		// Error code "success", no need to echo request back in this test
		Data: make([]byte, 4),
	}}

	c, tc := testConn(t, pid)
	tc.receive = [][]Message{replies}

	msgs, err := c.Execute(req)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	// Fill in fields for comparison
	req.Header.Length = uint32(nlmsgAlign(nlmsgLength(0)))
	req.Header.Sequence = 1
	req.Header.PID = pid

	if want, got := req, tc.send; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected request:\n- want: %#v\n-  got: %#v",
			want, got)
	}
	if want, got := replies, msgs; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected replies:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestConnSend(t *testing.T) {
	c, tc := testConn(t, 0)

	// Let Conn.Send populate length, sequence, PID
	m := Message{}

	out, err := c.Send(m)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// Make the same changes that Conn.Send should
	m = Message{
		Header: Header{
			Length:   uint32(nlmsgAlign(nlmsgLength(0))),
			Sequence: *c.seq,
		},
	}

	if want, got := tc.send, out; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected output message from Conn.Send:\n- want: %#v\n-  got: %#v",
			want, got)
	}
	if want, got := tc.send, m; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected modified message:\n- want: %#v\n-  got: %#v",
			want, got)
	}

	// Keep sending to verify sequence number increment
	seq := m.Header.Sequence
	for i := 0; i < 100; i++ {
		out, err := c.Send(Message{})
		if err != nil {
			t.Fatalf("failed to send message: %v", err)
		}

		seq++
		if want, got := seq, out.Header.Sequence; want != got {
			t.Fatalf("unexpected sequence number:\n- want: %v\n-  got: %v",
				want, got)
		}
	}
}

func TestConnReceiveMultiPartOnce(t *testing.T) {
	c, tc := testConn(t, 0)

	tc.receive = [][]Message{
		{
			{
				Header: Header{
					Flags: HeaderFlagsMulti,
				},
			},
		},
		{
			{
				Header: Header{
					Flags: HeaderFlagsMulti,
					Type:  HeaderTypeDone,
				},
			},
		},
	}

	msgs, err := c.Receive()
	if err != nil {
		t.Fatalf("failed to receive messages: %v", err)
	}

	if want, got := tc.receive[0], msgs; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected output messages from Conn.Receive:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestConnReceiveMultiPartRecursive(t *testing.T) {
	c, tc := testConn(t, 0)

	tc.receive = [][]Message{
		{
			{
				Header: Header{
					Flags: HeaderFlagsMulti,
				},
			},
		},
		{
			{
				Header: Header{
					Flags: HeaderFlagsMulti,
				},
			},
		},
		{
			{
				Header: Header{
					Flags: HeaderFlagsMulti,
				},
			},
		},
		{
			{
				Header: Header{
					Flags: HeaderFlagsMulti,
					Type:  HeaderTypeDone,
				},
			},
		},
	}

	msgs, err := c.Receive()
	if err != nil {
		t.Fatalf("failed to receive messages: %v", err)
	}

	want := append(tc.receive[0], append(tc.receive[1], tc.receive[2]...)...)
	if got := msgs; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected output messages from Conn.Receive:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestConnReceiveShortErrorMessage(t *testing.T) {
	c, tc := testConn(t, 0)
	tc.receive = [][]Message{{
		{
			Header: Header{
				Length: uint32(nlmsgAlign(nlmsgLength(4))),
				Type:   HeaderTypeError,
			},
			Data: []byte{0x01},
		},
	}}

	_, got := c.Receive()

	if want := errShortErrorMessage; want != got {
		t.Fatalf("unexpected error:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestConnJoinLeaveGroupUnsupported(t *testing.T) {
	want := errMulticastGroupsNotSupported

	c := newConn(&noopConn{}, 0)

	ops := []func(group uint32) error{
		c.JoinGroup,
		c.LeaveGroup,
	}

	for _, op := range ops {
		if got := op(0); want != got {
			t.Fatalf("unexpected error:\n- want: %v\n-  got %v",
				want, got)
		}
	}
}

func TestConnSetBPFUnsupported(t *testing.T) {
	want := errBPFFiltersNotSupported

	c := newConn(&noopConn{}, 0)

	if got := c.SetBPF(nil); want != got {
		t.Fatalf("unexpected error:\n- want: %v\n-  got %v",
			want, got)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		req  Message
		rep  []Message
		err  error
	}{
		{
			name: "mismatched sequence",
			req: Message{
				Header: Header{
					Sequence: 1,
				},
			},
			rep: []Message{{
				Header: Header{
					Sequence: 2,
				},
			}},
			err: errMismatchedSequence,
		},
		{
			name: "mismatched sequence second message",
			req: Message{
				Header: Header{
					Sequence: 1,
				},
			},
			rep: []Message{
				{
					Header: Header{
						Sequence: 1,
					},
				},
				{
					Header: Header{
						Sequence: 2,
					},
				},
			},
			err: errMismatchedSequence,
		},
		{
			name: "mismatched PID",
			req: Message{
				Header: Header{
					PID: 1,
				},
			},
			rep: []Message{{
				Header: Header{
					PID: 2,
				},
			}},
			err: errMismatchedPID,
		},
		{
			name: "mismatched PID second message",
			req: Message{
				Header: Header{
					PID: 1,
				},
			},
			rep: []Message{
				{
					Header: Header{
						PID: 1,
					},
				},
				{
					Header: Header{
						PID: 2,
					},
				},
			},
			err: errMismatchedPID,
		},
		{
			name: "OK matching sequence and PID",
			req: Message{
				Header: Header{
					Sequence: 1,
					PID:      1,
				},
			},
			rep: []Message{{
				Header: Header{
					Sequence: 1,
					PID:      1,
				},
			}},
		},
		{
			name: "OK multicast messages",
			// No request
			req: Message{},
			rep: []Message{{
				Header: Header{
					Sequence: 1,
					PID:      0,
				},
			}},
		},
		{
			name: "OK no PID assigned yet",
			// No request
			req: Message{
				Header: Header{
					Sequence: 1,
					PID:      0,
				},
			},
			rep: []Message{{
				Header: Header{
					Sequence: 1,
					PID:      9999,
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.req, tt.rep)

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

func testConn(t *testing.T, pid uint32) (*Conn, *testOSConn) {
	c := &testOSConn{}
	return newConn(c, pid), c
}

type testOSConn struct {
	send    Message
	receive [][]Message

	calls int

	noopConn
}

func (c *testOSConn) Send(m Message) error {
	c.send = m
	return nil
}

func (c *testOSConn) Receive() ([]Message, error) {
	defer func() { c.calls++ }()

	return c.receive[c.calls], nil
}

var _ osConn = &noopConn{}

type noopConn struct{}

func (c *noopConn) Close() error                { return nil }
func (c *noopConn) Send(m Message) error        { return nil }
func (c *noopConn) Receive() ([]Message, error) { return nil, nil }
