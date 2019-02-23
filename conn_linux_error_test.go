// +build linux

package netlink_test

import (
	"encoding/binary"
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"github.com/mdlayher/netlink/nltest"
	"golang.org/x/sys/unix"
)

func TestConnReceiveErrorLinux(t *testing.T) {
	skipBigEndian(t)

	// Note: using *Conn instead of Linux-only *conn, to test
	// error handling logic in *Conn.Receive

	tests := []struct {
		name string
		msgs []netlink.Message
		err  error
	}{
		{
			name: "ENOENT",
			msgs: []netlink.Message{{
				Header: netlink.Header{
					Length:   20,
					Type:     netlink.Error,
					Sequence: 1,
					PID:      1,
				},
				// -2, little endian (ENOENT)
				Data: []byte{0xfe, 0xff, 0xff, 0xff},
			}},
			err: unix.ENOENT,
		},
		{
			name: "multipart done without error attached",
			msgs: []netlink.Message{
				{
					Header: netlink.Header{
						Flags: netlink.Multi,
					},
				},
				{
					Header: netlink.Header{
						Type:  netlink.Done,
						Flags: netlink.Multi,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := nltest.Dial(func(_ []netlink.Message) ([]netlink.Message, error) {
				return tt.msgs, nil
			})
			defer c.Close()

			// Need to prepopulate nltest's internal buffers by invoking the
			// function once.
			_, _ = c.Send(netlink.Message{})

			_, err := c.Receive()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

func skipBigEndian(t *testing.T) {
	if nlenc.NativeEndian() == binary.BigEndian {
		t.Skip("skipping test on big-endian system")
	}
}
