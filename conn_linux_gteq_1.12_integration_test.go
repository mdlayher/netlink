//+build go1.12,linux

package netlink_test

import (
	"net"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jsimonetti/rtnetlink/rtnl"
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

func TestIntegrationConnNetNSExplicit(t *testing.T) {
	skipUnprivileged(t)

	// Create a network namespace for use within this test.
	const ns = "nltest0"
	shell(t, "ip", "netns", "add", ns)
	defer shell(t, "ip", "netns", "del", ns)

	f, err := os.Open("/var/run/netns/" + ns)
	if err != nil {
		t.Fatalf("failed to open namespace file: %v", err)
	}
	defer f.Close()

	// Create a connection in each the host namespace and the new network
	// namespace. We will use these to validate that a namespace was entered
	// and that an interface creation notification was only visible to the
	// connection within the namespace.
	hostC := rtnlDial(t, 0)
	defer hostC.Close()

	nsC := rtnlDial(t, int(f.Fd()))
	defer nsC.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	go func() {
		defer wg.Done()

		_, err := hostC.Receive()
		if err == nil {
			panic("received netlink message in host namespace")
		}

		// Timeout means we were interrupted, so return.
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			return
		}

		panicf("failed to receive in host namespace: %v", err)
	}()

	// Create a temporary interface within the new network namespace.
	const ifName = "nltestns0"
	defer shell(t, "ip", "netns", "exec", ns, "ip", "link", "del", ifName)

	ifi := rtnlReceive(t, nsC, func() {
		// Trigger a notification in the new namespace.
		shell(t, "ip", "netns", "exec", ns, "ip", "tuntap", "add", ifName, "mode", "tun")
	})

	// And finally interrupt the host connection so it can exit its
	// receive goroutine.
	if err := hostC.SetDeadline(time.Unix(1, 0)); err != nil {
		t.Fatalf("failed to interrupt host connection: %v", err)
	}

	if diff := cmp.Diff(ifName, ifi); diff != "" {
		t.Fatalf("unexpected interface name (-want +got):\n%s", diff)
	}
}

func TestIntegrationConnNetNSImplicit(t *testing.T) {
	skipUnprivileged(t)

	// Create a network namespace for use within this test.
	const ns = "nltest0"
	shell(t, "ip", "netns", "add", ns)
	defer shell(t, "ip", "netns", "del", ns)

	f, err := os.Open("/var/run/netns/" + ns)
	if err != nil {
		t.Fatalf("failed to open namespace file: %v", err)
	}
	defer f.Close()

	// Create an interface in the new namespace. We will attempt to find it later.
	const ifName = "nltestns0"
	shell(t, "ip", "netns", "exec", ns, "ip", "tuntap", "add", ifName, "mode", "tun")
	defer shell(t, "ip", "netns", "exec", ns, "ip", "link", "del", ifName)

	// We're going to manipulate the network namespace of this thread, so we
	// must lock OS thread and keep track of the original namespace for later.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origNS, err := netlink.GetThreadNetNS()
	if err != nil {
		t.Fatalf("failed to get current network namespace: %v", err)
	}

	defer func() {
		if err := netlink.SetThreadNetNS(origNS); err != nil {
			t.Fatalf("failed to restore original network namespace: %v", err)
		}
	}()

	if err := netlink.SetThreadNetNS(int(f.Fd())); err != nil {
		t.Fatalf("failed to enter new network namespace: %v", err)
	}

	// Any netlink connections created beyond this point should set themselves
	// into the new namespace automatically as well.

	c, err := rtnl.Dial(nil)
	if err != nil {
		t.Fatalf("failed to dial rtnetlink: %v", err)
	}
	defer c.Close()

	ifis, err := c.Links()
	if err != nil {
		t.Fatalf("failed to list links: %v", err)
	}

	var found bool
	for _, ifi := range ifis {
		if ifi.Name == ifName {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("did not find interface %q in namespace %q", ifName, ns)
	}
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
