//+build !go1.12,linux

package netlink

import "os"

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
