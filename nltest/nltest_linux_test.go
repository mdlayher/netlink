package nltest_test

import (
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nltest"
	"golang.org/x/sys/unix"
)

func TestLinuxDialError(t *testing.T) {
	c := nltest.Dial(func(req []netlink.Message) ([]netlink.Message, error) {
		return nltest.Error(int(unix.ENOENT), req)
	})

	if _, err := c.Execute(netlink.Message{}); !netlink.IsNotExist(err) {
		t.Fatalf("expected error is not exist, but got: %v", err)
	}
}
