//+build !tip

package netlink

import (
	"os"
	"syscall"
)

func setBlockingMode(sysfd int) error {
	return nil
}

func fdread(fd *os.File, f func(int) (done bool)) error {
	f(int(fd.Fd()))
	return nil
}

func fdwrite(fd *os.File, f func(int) (done bool)) error {
	f(int(fd.Fd()))
	return nil
}

func fdcontrol(fd *os.File, f func(int)) error {
	f(int(fd.Fd()))
	return nil
}

func newRawConn(fd *os.File) (syscall.RawConn, error) {
	return &rawConn{fd: fd.Fd()}, nil
}

var _ syscall.RawConn = &rawConn{}

// A rawConn is a syscall.RawConn.
type rawConn struct {
	fd uintptr
}

func (rc *rawConn) Control(f func(fd uintptr)) error {
	f(rc.fd)
	return nil
}

// TODO(mdlayher): implement Read and Write?

func (rc *rawConn) Read(_ func(fd uintptr) (done bool)) error  { return errSyscallConnNotSupported }
func (rc *rawConn) Write(_ func(fd uintptr) (done bool)) error { return errSyscallConnNotSupported }
