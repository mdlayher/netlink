package genltest_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"
	"github.com/mdlayher/netlink/genetlink/genltest"
)

func TestConnSend(t *testing.T) {
	req := genetlink.Message{
		Data: []byte{0xff, 0xff, 0xff, 0xff},
	}

	c := genltest.Dial(func(creq genetlink.Message, _ netlink.Message) ([]genetlink.Message, error) {
		if want, got := req.Data, creq.Data; !bytes.Equal(want, got) {
			t.Fatalf("unexpected request data:\n- want: %v\n-  got: %v",
				want, got)
		}

		return nil, nil
	})
	defer c.Close()

	if _, err := c.Send(req, 1, 1); err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
}

func TestConnExecuteOK(t *testing.T) {
	req := genetlink.Message{
		Data: []byte{0xff},
	}

	c := genltest.Dial(func(creq genetlink.Message, _ netlink.Message) ([]genetlink.Message, error) {
		// Turn the request back around to the client.
		return []genetlink.Message{creq}, nil
	})
	defer c.Close()

	got, err := c.Execute(req, 1, 1)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	if want := []genetlink.Message{req}; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected response messages:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestConnExecuteNoMessages(t *testing.T) {
	c := genltest.Dial(func(_ genetlink.Message, _ netlink.Message) ([]genetlink.Message, error) {
		return nil, io.EOF
	})
	defer c.Close()

	msgs, err := c.Execute(genetlink.Message{}, 0, 0)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if l := len(msgs); l > 0 {
		t.Fatalf("expected no generic netlink messages, but got: %d", l)
	}
}

func TestConnReceiveNoMessages(t *testing.T) {
	c := genltest.Dial(func(_ genetlink.Message, _ netlink.Message) ([]genetlink.Message, error) {
		return nil, io.EOF
	})
	defer c.Close()

	gmsgs, nmsgs, err := c.Receive()
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if l := len(gmsgs); l > 0 {
		t.Fatalf("expected no generic netlink messages, but got: %d", l)
	}

	if l := len(nmsgs); l > 0 {
		t.Fatalf("expected no netlink messages, but got: %d", l)
	}
}

func TestConnReceiveError(t *testing.T) {
	errFoo := errors.New("foo")

	c := genltest.Dial(func(_ genetlink.Message, _ netlink.Message) ([]genetlink.Message, error) {
		return nil, errFoo
	})
	defer c.Close()

	_, _, err := c.Receive()
	if err != errFoo {
		t.Fatalf("unexpected error: %v", err)
	}
}
