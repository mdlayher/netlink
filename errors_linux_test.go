// +build linux

package netlink_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
	errors "golang.org/x/xerrors" // for go<1.13
)

func TestIsNotExistLinux(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ENOBUFS",
			err:  unix.ENOBUFS,
		},
		{
			name: "OpError ENOBUFS",
			err: &netlink.OpError{
				Op:  "receive",
				Err: unix.ENOBUFS,
			},
		},
		{
			name: "OpError os.SyscallError ENOBUFS",
			err: &netlink.OpError{
				Op:  "receive",
				Err: os.NewSyscallError("recvmsg", unix.ENOBUFS),
			},
		},
		{
			name: "ENOENT",
			err:  unix.ENOENT,
			want: true,
		},
		{
			name: "OpError ENOENT",
			err: &netlink.OpError{
				Op:  "receive",
				Err: unix.ENOENT,
			},
			want: true,
		},
		{
			name: "OpError os.SyscallError ENOENT",
			err: &netlink.OpError{
				Op:  "receive",
				Err: os.NewSyscallError("recvmsg", unix.ENOENT),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := netlink.IsNotExist(tt.err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

var unwrapTestcases = []struct {
	err error
	in  error
	out bool
}{
	{err: os.ErrExist, in: nil, out: false},
	{err: os.ErrExist, in: &netlink.OpError{Op: "receive", Err: os.ErrExist}, out: true},
	{err: os.ErrExist, in: &netlink.OpError{Op: "receive", Err: os.ErrPermission}, out: false},
	{err: os.ErrExist, in: &netlink.OpError{Op: "receive", Err: os.ErrNotExist}, out: false},
	{err: os.ErrExist, in: os.ErrExist, out: true},
	{err: os.ErrExist, in: os.ErrPermission, out: false},
	{err: os.ErrNotExist, in: nil, out: false},
	{err: os.ErrNotExist, in: &netlink.OpError{Op: "receive", Err: os.ErrExist}, out: false},
	{err: os.ErrNotExist, in: &netlink.OpError{Op: "receive", Err: os.ErrPermission}, out: false},
	{err: os.ErrNotExist, in: &netlink.OpError{Op: "receive", Err: os.ErrNotExist}, out: true},
	{err: os.ErrNotExist, in: os.ErrExist, out: false},
	{err: os.ErrNotExist, in: os.ErrPermission, out: false},
	{err: os.ErrNotExist, in: os.ErrNotExist, out: true},
	{err: os.ErrExist, in: os.ErrNotExist, out: false},
}

func TestUnwrapLinux(t *testing.T) {
	for i, tc := range unwrapTestcases {
		out := errors.Is(tc.in, tc.err)
		if out != tc.out {
			t.Errorf("Is(%q, %q) (#%d): expected %#v, got %#v", tc.in, tc.err, i, tc.out, out)
		}
	}
}
