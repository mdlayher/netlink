package netlink

import (
	"os"
	"reflect"
	"testing"
)

func TestConnExecute(t *testing.T) {
	req := Message{
		Header: Header{
			Flags: HeaderFlagsRequest | HeaderFlagsAcknowledge,
		},
	}

	replies := []Message{{
		Header: Header{
			Type:     HeaderTypeError,
			Sequence: 1,
			PID:      uint32(os.Getpid()),
		},
		// Error code "success", no need to echo request back in this test
		Data: make([]byte, 4),
	}}

	c, tc := testConn(t)
	tc.receive = replies

	msgs, err := c.Execute(req)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	// Fill in fields for comparison
	req.Header.Length = uint32(nlmsgAlign(nlmsgLength(0)))
	req.Header.Sequence = 1
	req.Header.PID = uint32(os.Getpid())

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
	c, tc := testConn(t)

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
			Sequence: 1,
			PID:      uint32(os.Getpid()),
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

func TestConnReceive(t *testing.T) {
	c, tc := testConn(t)
	tc.receive = []Message{
		{
			Header: Header{
				Length:   uint32(nlmsgAlign(nlmsgLength(4))),
				Sequence: 1,
				PID:      uint32(os.Getpid()),
			},
			Data: []byte{0x00, 0x11, 0x22, 0x33},
		},
		{
			Header: Header{
				Length:   uint32(nlmsgAlign(nlmsgLength(4))),
				Sequence: 1,
				PID:      uint32(os.Getpid()),
			},
			Data: []byte{0x44, 0x55, 0x66, 0x77},
		},
	}

	msgs, err := c.Receive()
	if err != nil {
		t.Fatalf("failed to receive messages: %v", err)
	}

	if want, got := tc.receive, msgs; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected output messages from Conn.Receive:\n- want: %#v\n-  got: %#v",
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
			name: "mismatched sequence",
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
			name: "short error message",
			req: Message{
				Header: Header{
					Sequence: 1,
					PID:      1,
				},
			},
			rep: []Message{
				{
					Header: Header{
						Sequence: 1,
						PID:      1,
					},
				},
				{
					Header: Header{
						Type:     HeaderTypeError,
						Sequence: 1,
						PID:      1,
					},
					// One byte short of error
					Data: []byte{0x00, 0x11, 0x22},
				},
			},
			err: errShortErrorMessage,
		},
		{
			name: "OK success",
			req: Message{
				Header: Header{
					Sequence: 1,
					PID:      1,
				},
			},
			rep: []Message{{
				Header: Header{
					Type:     HeaderTypeError,
					Sequence: 1,
					PID:      1,
				},
				// Success
				Data: make([]byte, 4),
			}},
		},
		{
			name: "OK header type noop",
			req: Message{
				Header: Header{
					Sequence: 1,
					PID:      1,
				},
			},
			rep: []Message{{
				Header: Header{
					Type:     HeaderTypeNoop,
					Sequence: 1,
					PID:      1,
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

func testConn(t *testing.T) (*Conn, *testOSConn) {
	c := &testOSConn{}
	return newConn(c), c
}

type testOSConn struct {
	send    Message
	receive []Message

	noopConn
}

func (c *testOSConn) Send(m Message) error {
	c.send = m
	return nil
}

func (c *testOSConn) Receive() ([]Message, error) {
	return c.receive, nil
}

type noopConn struct{}

func (c *noopConn) Close() error                 { return nil }
func (c *noopConn) Send(m Message) error         { return nil }
func (c *noopConn) Receives() ([]Message, error) { return nil, nil }
