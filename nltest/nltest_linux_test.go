package nltest_test

import (
	"os"
	"syscall"
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nltest"
)

func TestLinuxDialError(t *testing.T) {
	c := nltest.Dial(func(req netlink.Message) ([]netlink.Message, error) {
		return nltest.Error(int(syscall.ENOENT), req)
	})

	if _, err := c.Execute(netlink.Message{}); !os.IsNotExist(err) {
		t.Fatalf("expected error is not exist, but got: %v", err)
	}
}
