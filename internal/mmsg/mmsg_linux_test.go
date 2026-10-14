//go:build linux

package mmsg_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/socket"
	"golang.org/x/sys/unix"

	"github.com/mdlayher/netlink/internal/mmsg"
)

func TestConnReceive(t *testing.T) {
	s, peer := socketPair(t)
	c := mmsg.New(s, mmsg.Config{Messages: 2, BufferSize: 16})

	send(t, peer, "a", "bb", "ccc")

	// The first receive fills the batch, the second gets the rest.
	for _, want := range [][]string{{"a", "bb"}, {"ccc"}} {
		dgs, err := c.Receive(context.Background())
		if err != nil {
			t.Fatalf("failed to receive: %v", err)
		}

		var got []string
		for _, d := range dgs {
			if d.Truncated {
				t.Fatalf("unexpected truncated datagram: %q", d.Data)
			}
			got = append(got, string(d.Data))
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("unexpected datagrams (-want +got):\n%s", diff)
		}
	}
}

func TestConnTruncated(t *testing.T) {
	s, peer := socketPair(t)
	c := mmsg.New(s, mmsg.Config{Messages: 2, BufferSize: 4})

	send(t, peer, "hello", "hi")

	dgs, err := c.Receive(context.Background())
	if err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	want := []mmsg.Datagram{
		{Data: []byte("hell"), Truncated: true},
		{Data: []byte("hi")},
	}

	if diff := cmp.Diff(want, dgs); diff != "" {
		t.Fatalf("unexpected datagrams (-want +got):\n%s", diff)
	}
}

func TestConnContext(t *testing.T) {
	s, _ := socketPair(t)
	c := mmsg.New(s, mmsg.Config{Messages: 2, BufferSize: 16})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Nothing has been sent, so Receive should block until the context expires.
	_, err := c.Receive(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, but got: %v", err)
	}
}

// socketPair returns a connected pair of Unix datagram sockets: one wrapped in
// a *socket.Conn for receiving, and the raw file descriptor of its peer.
func socketPair(t *testing.T) (*socket.Conn, int) {
	t.Helper()

	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatalf("failed to create socket pair: %v", err)
	}

	c, err := socket.New(fds[0], "mmsg")
	if err != nil {
		t.Fatalf("failed to wrap socket: %v", err)
	}

	t.Cleanup(func() {
		_ = c.Close()
		_ = unix.Close(fds[1])
	})

	return c, fds[1]
}

// send writes each string to fd as its own datagram.
func send(t *testing.T, fd int, datagrams ...string) {
	t.Helper()

	for _, d := range datagrams {
		if _, err := unix.Write(fd, []byte(d)); err != nil {
			t.Fatalf("failed to send datagram: %v", err)
		}
	}
}
