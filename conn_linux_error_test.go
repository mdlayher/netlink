// +build linux

package netlink_test

import (
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nltest"
	"golang.org/x/sys/unix"
)

func TestConnReceiveErrorLinux(t *testing.T) {
	// Note: using *Conn instead of Linux-only *conn, to test
	// error handling logic in *Conn.Receive

	tests := []struct {
		name  string
		multi bool
		msgs  []netlink.Message
		err   error
	}{
		{
			name: "ENOENT",
			msgs: []netlink.Message{{
				Header: netlink.Header{
					Length:   20,
					Type:     netlink.HeaderTypeError,
					Sequence: 1,
					PID:      1,
				},
				// -2, little endian (ENOENT)
				Data: []byte{0xfe, 0xff, 0xff, 0xff},
			}},
			err: unix.ENOENT,
		},
		{
			name: "EINTR multipart",
			msgs: []netlink.Message{
				{
					Header: netlink.Header{
						Flags: netlink.HeaderFlagsMulti,
					},
				},
				{
					Header: netlink.Header{
						Type:  netlink.HeaderTypeError,
						Flags: netlink.HeaderFlagsMulti,
					},
					Data: []byte{0xfc, 0xff, 0xff, 0xff},
				},
			},
			// -4, little endian (EINTR)
			err: unix.EINTR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := nltest.Dial(func(_ netlink.Message) ([]netlink.Message, error) {
				if tt.multi {
					return nltest.Multipart(tt.msgs)
				}

				return tt.msgs, nil
			})
			defer c.Close()

			_, err := c.Receive()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}
