//+build go1.12,integration,linux

package netlink_test

import (
	"net"
	"testing"
	"time"

	"github.com/mdlayher/netlink"

	"golang.org/x/sys/unix"
)

func TestIntegrationConnTimeout(t *testing.T) {
	conn, err := netlink.Dial(unix.NETLINK_GENERIC, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	timeout := 1 * time.Millisecond
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("failed to set deadline: %v", err)
	}

	errC := make(chan error)
	go func() {
		_, err := conn.Receive()
		errC <- err
	}()

	select {
	case err := <-errC:
		mustBeTimeoutNetError(t, err)
	case <-time.After(timeout + 1*time.Millisecond):
		t.Fatalf("timeout did not fire")
	}
}

func TestIntegrationConnExecuteAfterReadDeadline(t *testing.T) {
	conn, err := netlink.Dial(unix.NETLINK_GENERIC, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	timeout := 1 * time.Millisecond
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("failed to set deadline: %v", err)
	}
	time.Sleep(2 * timeout)

	req := netlink.Message{
		Header: netlink.Header{
			Flags:    netlink.Request | netlink.Acknowledge,
			Sequence: 1,
		},
	}
	got, err := conn.Execute(req)
	if err == nil {
		t.Fatalf("Execute succeeded: got %v", got)
	}
	mustBeTimeoutNetError(t, err)
}

func mustBeTimeoutNetError(t *testing.T, err error) {
	t.Helper()
	ne, ok := err.(net.Error)
	if !ok {
		t.Fatalf("didn't get a net.Error: got a %T instead", err)
	}
	if !ne.Timeout() {
		t.Fatalf("didn't get a timeout")
	}
}
