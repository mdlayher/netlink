package nltest_test

import (
	"os"
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

func TestLinuxSyscallError(t *testing.T) {
	c := nltest.Dial(func(req []netlink.Message) ([]netlink.Message, error) {
		return nil, unix.ENOENT
	})

	_, err := c.Execute(netlink.Message{})
	if !netlink.IsNotExist(err) {
		t.Fatalf("expected error is not exist, but got: %v", err)
	}

	// Expect raw system call errors to be wrapped.
	_ = err.(*netlink.OpError).Err.(*os.SyscallError)
}
