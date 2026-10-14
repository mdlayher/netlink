//go:build linux && !386 && !s390x

package mmsg

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// recvmmsg invokes recvmmsg(2) on fd to fill hdrs, with no timeout, and
// returns the number of messages received.
func recvmmsg(fd int, hdrs []unix.Mmsghdr, flags int) (int, unix.Errno) {
	n, _, errno := unix.Syscall6(
		unix.SYS_RECVMMSG,
		uintptr(fd),
		uintptr(unsafe.Pointer(&hdrs[0])),
		uintptr(len(hdrs)),
		uintptr(flags),
		0,
		0,
	)

	return int(n), errno
}
