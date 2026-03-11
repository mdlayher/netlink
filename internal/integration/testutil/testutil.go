//go:build linux
// +build linux

package testutil

import (
	"os"
	"runtime"
	"testing"

	"github.com/google/nftables"
	"github.com/vishvananda/netns"
)

// SkipUnprivileged skips the test if the current user is not root.
func SkipUnprivileged(t testing.TB) {
	t.Helper()
	if uid := os.Getuid(); uid != 0 {
		t.Skipf("skipping test: requires root privileges")
	}
}

// NewNS creates a new network namespace and returns its handle, along with a
// cleanup function to close it.
func NewNS(t testing.TB) (netns.NsHandle, func()) {
	t.Helper()
	SkipUnprivileged(t)

	// Locking the thread is necessary to ensure that the namespace created by
	// `netns.New()` is correctly associated with the current goroutine.
	runtime.LockOSThread()

	ns, err := netns.New()
	if err != nil {
		t.Fatalf("netns.New() failed: %v", err)
	}

	closer := func() {
		runtime.UnlockOSThread()
		if err := ns.Close(); err != nil {
			t.Fatalf("ns.Close() failed: %v", err)
		}
	}

	return ns, closer
}

// NewNftablesConn creates a new nftables connection within the specified
// network namespace.
func NewNftablesConn(t testing.TB, ns netns.NsHandle) *nftables.Conn {
	t.Helper()
	SkipUnprivileged(t)

	conn, err := nftables.New(nftables.WithNetNSFd(int(ns)))
	if err != nil {
		t.Fatalf("failed to create nftables connection: %v", err)
	}

	return conn
}
