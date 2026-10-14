//go:build linux

package mmsg

import (
	"context"
	"unsafe"

	"github.com/mdlayher/socket"
	"golang.org/x/sys/unix"
)

// A Datagram is a datagram returned by Conn.Receive.
type Datagram struct {
	// Data holds the contents of the datagram. It is only valid until the
	// next call to Receive.
	Data []byte

	// Truncated reports whether the datagram was larger than its buffer, in
	// which case Data only holds the start of it.
	Truncated bool
}

// Config contains options for a Conn.
type Config struct {
	// Messages is the maximum number of datagrams returned by each call to
	// Receive. It must be positive.
	Messages int

	// BufferSize is the size in bytes of the buffer for each datagram.
	// Datagrams larger than BufferSize are truncated. It must be positive.
	BufferSize int
}

// A Conn receives multiple datagrams from a socket with a single recvmmsg(2)
// call, reusing the same buffers for every call.
type Conn struct {
	s    *socket.Conn
	bufs [][]byte
	iovs []unix.Iovec
	hdrs []unix.Mmsghdr
	out  []Datagram
}

// New creates a Conn which receives datagrams from s using the options in
// cfg.
func New(s *socket.Conn, cfg Config) *Conn {
	n := cfg.Messages
	c := &Conn{
		s:    s,
		bufs: make([][]byte, n),
		iovs: make([]unix.Iovec, n),
		hdrs: make([]unix.Mmsghdr, n),
		out:  make([]Datagram, n),
	}

	for i := range n {
		c.bufs[i] = make([]byte, cfg.BufferSize)
		c.iovs[i].Base = &c.bufs[i][0]
		c.iovs[i].SetLen(cfg.BufferSize)
		c.hdrs[i].Hdr.Iov = &c.iovs[i]
		c.hdrs[i].Hdr.SetIovlen(1)
	}

	return c
}

// Receive receives up to Config.Messages datagrams, blocking until at least
// one is available. The returned slice is only valid until the next call to
// Receive.
func (c *Conn) Receive(ctx context.Context) ([]Datagram, error) {
	var n int
	err := c.s.ReadFunc(ctx, "recvmmsg", func(fd int) error {
		rn, _, errno := unix.Syscall6(
			unix.SYS_RECVMMSG,
			uintptr(fd),
			uintptr(unsafe.Pointer(&c.hdrs[0])),
			uintptr(len(c.hdrs)),
			unix.MSG_WAITFORONE,
			0,
			0,
		)
		if errno != 0 {
			return errno
		}

		n = int(rn)
		return nil
	})
	if err != nil {
		return nil, err
	}

	for i := range n {
		h := &c.hdrs[i]
		c.out[i] = Datagram{
			Data:      c.bufs[i][:h.Len],
			Truncated: h.Hdr.Flags&unix.MSG_TRUNC != 0,
		}
	}

	return c.out[:n], nil
}
