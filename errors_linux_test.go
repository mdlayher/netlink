// +build linux

package netlink_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
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

func TestIsExistLinux(t *testing.T) {
	var testcases = []struct {
		in  error
		out bool
	}{
		{in: nil, out: false},
		{in: &netlink.OpError{Op: "receive", Err: os.ErrExist}, out: true},
		{in: &netlink.OpError{Op: "receive", Err: os.ErrPermission}, out: false},
		{in: &netlink.OpError{Op: "receive", Err: os.ErrNotExist}, out: false},
		{in: os.ErrExist, out: true},
		{in: os.ErrPermission, out: false},
		{in: os.ErrNotExist, out: false},
	}
	for i, tc := range testcases {
		out := netlink.IsExist(tc.in)
		if out != tc.out {
			t.Errorf("'%v' (#%d): expected %#v, got %#v", tc.in, i, tc.out, out)
		}
	}
}
