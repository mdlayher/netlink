//+build tip

package netlink

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func setBlockingMode(sysfd int) error {
	return unix.SetNonblock(sysfd, true)
}

func fdread(fd *os.File, f func(int) (done bool)) error {
	rc, err := fd.SyscallConn()
	if err != nil {
		return err
	}
	return rc.Read(func(sysfd uintptr) bool {
		return f(int(sysfd))
	})
}

func fdwrite(fd *os.File, f func(int) (done bool)) error {
	rc, err := fd.SyscallConn()
	if err != nil {
		return err
	}
	return rc.Write(func(sysfd uintptr) bool {
		return f(int(sysfd))
	})
}

func fdcontrol(fd *os.File, f func(int)) error {
	rc, err := fd.SyscallConn()
	if err != nil {
		return err
	}
	return rc.Control(func(sysfd uintptr) {
		f(int(sysfd))
	})
}

func newRawConn(fd *os.File) (syscall.RawConn, error) {
	return fd.SyscallConn()
}
