package genltest_test

import (
	"bytes"
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
