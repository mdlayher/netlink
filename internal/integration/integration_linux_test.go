//go:build linux
// +build linux

package integration_test

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jsimonetti/rtnetlink"
	"github.com/mdlayher/ethtool"
	"github.com/mdlayher/netlink"
	"golang.org/x/net/nettest"
	"golang.org/x/sys/unix"
)

func TestIntegrationConnMulticast(t *testing.T) {
	skipUnprivileged(t)

	c, done := rtnlDial(t, 0)
	defer done()

	// Create an interface to trigger a notification, and remove it at the end
	// of the test.
	const ifName = "nltest0"
	defer shell(t, "ip", "link", "del", ifName)

	ifi := rtnlReceive(t, c, func() {
		shell(t, "ip", "tuntap", "add", ifName, "mode", "tun")
	})

	if diff := cmp.Diff(ifName, ifi); diff != "" {
		t.Fatalf("unexpected interface name (-want +got):\n%s", diff)
	}
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
	hostC, hostDone := rtnlDial(t, 0)
	defer hostDone()

	nsC, nsDone := rtnlDial(t, int(f.Fd()))
	defer nsDone()

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

func TestIntegrationRTNetlinkStrictCheckExtendedAcknowledge(t *testing.T) {
	c, err := netlink.Dial(unix.NETLINK_ROUTE, nil)
	if err != nil {
		t.Fatalf("failed to open rtnetlink socket: %s", err)
	}
	defer c.Close()

	// Turn on extended acknowledgements and strict checking so rtnetlink
	// reports detailed error information regarding our invalid dump request.
	setStrictCheck(t, c)
	if err := c.SetOption(netlink.ExtendedAcknowledge, true); err != nil {
		t.Fatalf("failed to set extended acknowledge option: %v", err)
	}

	// The kernel will complain that this field isn't valid for a filtered dump
	// request.
	b, err := (&rtnetlink.RouteMessage{SrcLength: 1}).MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	_, err = c.Execute(netlink.Message{
		Header: netlink.Header{
			Type:  unix.RTM_GETROUTE,
			Flags: netlink.Request | netlink.Dump,
		},
		Data: b,
	})

	oerr, ok := err.(*netlink.OpError)
	if !ok {
		t.Fatalf("expected *netlink.OpError, but got: %T", err)
	}

	// Assume the message contents will be relatively static but don't hardcode
	// offset just in case things change.

	want := &netlink.OpError{
		Op:      "receive",
		Err:     unix.EINVAL,
		Message: "Invalid values in header for FIB dump request",
	}

	if diff := cmp.Diff(want, oerr); diff != "" {
		t.Fatalf("unexpected *netlink.OpError (-want +got):\n%s", diff)
	}
}

func TestIntegrationRTNetlinkRouteManipulation(t *testing.T) {
	skipUnprivileged(t)

	c, err := netlink.Dial(unix.NETLINK_ROUTE, nil)
	if err != nil {
		t.Fatalf("failed to open rtnetlink socket: %s", err)
	}
	defer c.Close()

	// Required for kernel route dump filtering.
	setStrictCheck(t, c)

	lo, err := nettest.LoopbackInterface()
	if err != nil {
		t.Fatalf("failed to get loopback: %v", err)
	}

	// Install synthetic routes in documentation ranges into a non-default table
	// which we will later dump.
	const (
		table   = 100
		ip4Mask = 32
		ip6Mask = 128
	)

	var (
		ip4 = &net.IPNet{
			IP:   net.IPv4(192, 2, 2, 1),
			Mask: net.CIDRMask(ip4Mask, ip4Mask),
		}

		ip6 = &net.IPNet{
			IP:   net.ParseIP("2001:db8::1"),
			Mask: net.CIDRMask(ip6Mask, ip6Mask),
		}

		want = []*net.IPNet{ip4, ip6}
	)

	rtmsgs := []rtnetlink.RouteMessage{
		{
			Family:    unix.AF_INET,
			DstLength: ip4Mask,
			Protocol:  unix.RTPROT_STATIC,
			Scope:     unix.RT_SCOPE_UNIVERSE,
			Type:      unix.RTN_UNICAST,
			Attributes: rtnetlink.RouteAttributes{
				Dst:      ip4.IP,
				OutIface: uint32(lo.Index),
				Table:    table,
			},
		},
		{
			Family:    unix.AF_INET6,
			DstLength: ip6Mask,
			Protocol:  unix.RTPROT_STATIC,
			Scope:     unix.RT_SCOPE_UNIVERSE,
			Type:      unix.RTN_UNICAST,
			Attributes: rtnetlink.RouteAttributes{
				Dst:      ip6.IP,
				OutIface: uint32(lo.Index),
				Table:    table,
			},
		},
	}

	// Verify we can send a batch of updates in one syscall.
	var msgs []netlink.Message
	for _, m := range rtmsgs {
		b, err := m.MarshalBinary()
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		msgs = append(msgs, netlink.Message{
			Header: netlink.Header{
				Type:  unix.RTM_NEWROUTE,
				Flags: netlink.Request | netlink.Create | netlink.Replace,
			},
			Data: b,
		})
	}

	if _, err := c.SendMessages(msgs); err != nil {
		t.Fatalf("failed to add routes: %v", err)
	}

	// Only dump routes from the specified table.
	b, err := (&rtnetlink.RouteMessage{
		Attributes: rtnetlink.RouteAttributes{Table: table},
	}).MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	routes, err := c.Execute(
		netlink.Message{
			Header: netlink.Header{
				Type:  unix.RTM_GETROUTE,
				Flags: netlink.Request | netlink.Dump,
			},
			Data: b,
		},
	)
	if err != nil {
		t.Fatalf("failed to dump routes: %v", err)
	}

	// Parse the routes back to Go structures.
	got := make([]*net.IPNet, 0, len(routes))
	for _, r := range routes {
		var rtm rtnetlink.RouteMessage
		if err := rtm.UnmarshalBinary(r.Data); err != nil {
			t.Fatalf("failed to unmarshal route: %v", err)
		}

		got = append(got, &net.IPNet{
			IP:   rtm.Attributes.Dst,
			Mask: net.CIDRMask(int(rtm.DstLength), int(rtm.DstLength)),
		})
	}

	// Now clear the routes and verify they're removed before ensuring we got
	// the expected routes.
	for i := range msgs {
		msgs[i].Header.Type = unix.RTM_DELROUTE
		msgs[i].Header.Flags = netlink.Request | netlink.Acknowledge
	}

	if _, err := c.SendMessages(msgs); err != nil {
		t.Fatalf("failed to send: %v", err)
	}
	if _, err := c.Receive(); err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected routes (-want +got):\n%s", diff)
	}
}

func TestIntegrationEthtoolExtendedAcknowledge(t *testing.T) {
	t.Parallel()

	// The ethtool package uses extended acknowledgements and should populate
	// all of netlink.OpError's fields when unwrapped.
	c, err := ethtool.New()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("skipping, ethtool genetlink not available on this system")
		}

		t.Fatalf("failed to open ethtool genetlink: %v", err)
	}

	_, err = c.LinkInfo(ethtool.Interface{Name: "notexist0"})
	if err == nil {
		t.Fatal("expected an error, but none occurred")
	}

	var oerr *netlink.OpError
	if !errors.As(err, &oerr) {
		t.Fatalf("expected wrapped *netlink.OpError, but got: %T", err)
	}

	// Assume the message contents will be relatively static but don't hardcode
	// offset just in case things change.
	if oerr.Offset == 0 {
		t.Fatal("no offset specified in *netlink.OpError")
	}
	oerr.Offset = 0

	want := &netlink.OpError{
		Op:      "receive",
		Err:     unix.ENODEV,
		Message: "no device matches name",
	}

	if diff := cmp.Diff(want, oerr); diff != "" {
		t.Fatalf("unexpected *netlink.OpError (-want +got):\n%s", diff)
	}
}

func rtnlDial(t *testing.T, netNS int) (*netlink.Conn, func()) {
	t.Helper()

	timer := time.AfterFunc(10*time.Second, func() {
		panic("test took too long")
	})

	c, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{
		Groups: unix.RTMGRP_LINK,
		NetNS:  netNS,
	})
	if err != nil {
		t.Fatalf("failed to dial rtnetlink: %v", err)
	}

	return c, func() {
		if err := c.Close(); err != nil {
			t.Fatalf("failed to close rtnetlink connection: %v", err)
		}

		// Stop the timer to prevent a panic if other tests run for a long time.
		timer.Stop()
	}
}

func setStrictCheck(t *testing.T, c *netlink.Conn) {
	if err := c.SetOption(netlink.GetStrictCheck, true); err != nil {
		if errors.Is(err, unix.ENOPROTOOPT) {
			t.Skipf("skipping, netlink strict checking is not supported on this kernel")
		}

		t.Fatalf("failed to set strict check option: %v", err)
	}
}

func rtnlReceive(t *testing.T, c *netlink.Conn, do func()) string {
	t.Helper()

	// Receive messages in goroutine.
	msgC := make(chan rtnetlink.LinkMessage)
	go func() {
		msgs, err := c.Receive()
		if err != nil {
			panicf("failed to receive rtnetlink messages: %v", err)
		}

		var rtmsg rtnetlink.LinkMessage
		if err := rtmsg.UnmarshalBinary(msgs[0].Data); err != nil {
			panicf("failed to unmarshal rtnetlink message: %v", err)
		}

		msgC <- rtmsg
	}()

	// Execute the function which will generate messages, and then wait for
	// a message.
	do()
	m := <-msgC

	return m.Attributes.Name
}

func skipUnprivileged(t *testing.T) {
	const ifName = "nlprobe0"
	shell(t, "ip", "tuntap", "add", ifName, "mode", "tun")
	shell(t, "ip", "link", "del", ifName)
}

func shell(t *testing.T, name string, arg ...string) {
	t.Helper()

	t.Logf("$ %s %v", name, arg)

	cmd := exec.Command(name, arg...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command %q: %v", name, err)
	}

	if err := cmd.Wait(); err != nil {
		// Shell operations in these tests require elevated privileges.
		if cmd.ProcessState.ExitCode() == int(unix.EPERM) {
			t.Skipf("skipping, permission denied: %v", err)
		}

		t.Fatalf("failed to wait for command %q: %v", name, err)
	}
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
