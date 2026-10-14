//go:build linux && (386 || s390x)

package mmsg

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// https://github.com/golang/sys/blob/master/unix/syscall_linux_386.go#L139
const _RECVMMSG = 19

// recvmmsg invokes recvmmsg(2) on fd to fill hdrs, with no timeout, and
// returns the number of messages received.
//
// golang.org/x/sys/unix makess socket calls on 386 and s390x, so we do the
// same here.
func recvmmsg(fd int, hdrs []unix.Mmsghdr, flags int) (int, unix.Errno) {
	args := [5]uintptr{
		uintptr(fd),
		uintptr(unsafe.Pointer(&hdrs[0])),
		uintptr(len(hdrs)),
		uintptr(flags),
		0,
	}

	n, _, errno := unix.Syscall(
		unix.SYS_SOCKETCALL,
		_RECVMMSG,
		uintptr(unsafe.Pointer(&args)),
		0,
	)

	return int(n), errno
}
