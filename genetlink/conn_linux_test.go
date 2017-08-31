//+build linux

package genetlink_test

import (
	"encoding"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"
	"github.com/mdlayher/netlink/genetlink/genltest"
	"github.com/mdlayher/netlink/nltest"
	"golang.org/x/sys/unix"
)

func TestConnExecute(t *testing.T) {
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: 1,
			Version: 1,
		},
	}

	wantnl := netlink.Message{
		Header: netlink.Header{
			Length: 20,
			Type:   unix.GENL_ID_CTRL,
			Flags:  netlink.HeaderFlagsRequest,
			PID:    nltest.PID,
		},
		Data: mustMarshal(req),
	}

	wantgenl := []genetlink.Message{{
		Header: genetlink.Header{
			Command: 1,
			Version: 1,
		},
		Data: []byte{0x01, 0x02, 0x03, 0x04},
	}}

	c := genltest.Dial(func(_ genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		if diff := diffNetlinkMessages(wantnl, nreq); diff != "" {
			t.Fatalf("unexpected sent netlink message (-want +got):\n%s", diff)
		}

		return wantgenl, nil
	})

	msgs, err := c.Execute(req, unix.GENL_ID_CTRL, netlink.HeaderFlagsRequest)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if diff := cmp.Diff(wantgenl, msgs); diff != "" {
		t.Fatalf("unexpected replies (-want +got):\n%s", diff)
	}
}

func TestConnSend(t *testing.T) {
	const (
		length = 24
		family = unix.GENL_ID_CTRL
		flags  = netlink.HeaderFlagsRequest
	)

	req := genetlink.Message{
		Header: genetlink.Header{
			Command: 1,
			Version: 1,
		},
		Data: []byte{0x00, 0x01, 0x02, 0x03},
	}

	want := netlink.Message{
		Header: netlink.Header{
			Length: length,
			Type:   family,
			Flags:  flags,
			PID:    nltest.PID,
		},
		Data: mustMarshal(req),
	}

	c := genltest.Dial(func(_ genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		if diff := diffNetlinkMessages(want, nreq); diff != "" {
			t.Fatalf("unexpected sent netlink message (-want +got):\n%s", diff)
		}

		return nil, nil
	})

	nlreq, err := c.Send(req, family, flags)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	if diff := diffNetlinkMessages(want, nlreq); diff != "" {
		t.Fatalf("unexpected returned netlink message (-want +got):\n%s", diff)
	}
}

func TestConnReceive(t *testing.T) {
	gmsgs := []genetlink.Message{
		{
			Header: genetlink.Header{
				Command: 1,
				Version: 1,
			},
			Data: make([]byte, 0),
		},
		{
			Header: genetlink.Header{
				Command: 2,
				Version: 1,
			},
			Data: []byte{
				0x01, 0x02, 0x03, 0x04,
			},
		},
	}

	c := genltest.Dial(func(_ genetlink.Message, _ netlink.Message) ([]genetlink.Message, error) {
		return gmsgs, nil
	})

	msgs, _, err := c.Receive()
	if err != nil {
		t.Fatalf("failed to receive messages: %v", err)
	}

	if diff := cmp.Diff(gmsgs, msgs); diff != "" {
		t.Fatalf("unexpected replies (-want +got):\n%s", diff)
	}
}

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

// diffNetlinkMessages compares two netlink.Messages after zeroing their
// sequence number fields that make equality checks in testing difficult.
func diffNetlinkMessages(want, got netlink.Message) string {
	want.Header.Sequence = 0
	got.Header.Sequence = 0

	return cmp.Diff(want, got)
}
