//+build linux

package netlink

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/sys/unix"
)

func TestLinuxConn_bindOK(t *testing.T) {
	s := &testSocket{}
	if _, _, err := bind(s, &Config{}); err != nil {
		t.Fatalf("failed to bind: %v", err)
	}

	addr := &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
	}

	if want, got := addr, s.bind; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected bind address:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestLinuxConn_bindBindErrorCloseSocket(t *testing.T) {
	// Trigger an error during bind with Bind, meaning that the socket should be
	// closed to avoid leaking file descriptors.
	s := &testSocket{
		bindErr: errors.New("cannot bind"),
	}

	if _, _, err := bind(s, &Config{}); err == nil {
		t.Fatal("no error occurred, but expected one")
	}

	if want, got := true, s.closed; want != got {
		t.Fatalf("unexpected socket closed:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestLinuxConn_bindGetsocknameErrorCloseSocket(t *testing.T) {
	// Trigger an error during bind with Getsockname, meaning that the socket
	// should be closed to avoid leaking file descriptors.
	s := &testSocket{
		getsocknameErr: errors.New("cannot get socket name"),
	}

	if _, _, err := bind(s, &Config{}); err == nil {
		t.Fatal("no error occurred, but expected one")
	}

	if want, got := true, s.closed; want != got {
		t.Fatalf("unexpected socket closed:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestLinuxConnSend(t *testing.T) {
	c, s := testLinuxConn(t, nil)

	req := Message{
		Header: Header{
			Length:   uint32(nlmsgAlign(nlmsgLength(2))),
			Flags:    HeaderFlagsRequest | HeaderFlagsAcknowledge,
			Sequence: 1,
			PID:      uint32(os.Getpid()),
		},
		Data: []byte{0x01, 0x02},
	}

	if err := c.Send(req); err != nil {
		t.Fatalf("error while sending: %v", err)
	}

	// Pad data to 4 bytes as is done when marshaling for later comparison
	req.Data = append(req.Data, 0x00, 0x00)

	to := &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
	}

	if want, got := 0, s.sendmsg.flags; want != got {
		t.Fatalf("unexpected sendmsg flags:\n- want: %v\n-  got: %v",
			want, got)
	}
	if want, got := to, s.sendmsg.to; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected sendmsg address:\n- want: %#v\n-  got: %#v",
			want, got)
	}

	var out Message
	if err := (&out).UnmarshalBinary(s.sendmsg.p); err != nil {
		t.Fatalf("failed to unmarshal sendmsg buffer into message: %v", err)
	}

	if want, got := req, out; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected output message:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestLinuxConnReceiveInvalidSockaddr(t *testing.T) {
	c, s := testLinuxConn(t, nil)

	s.recvmsg.from = &unix.SockaddrInet4{}

	_, got := c.Receive()
	if want := errInvalidSockaddr; want != got {
		t.Fatalf("unexpected error:\n-  want: %v\n-  got: %v", want, got)
	}
}

func TestLinuxConnReceiveInvalidFamily(t *testing.T) {
	c, s := testLinuxConn(t, nil)

	s.recvmsg.from = &unix.SockaddrNetlink{
		// Should always be AF_NETLINK
		Family: unix.AF_NETLINK + 1,
	}

	_, got := c.Receive()
	if want := errInvalidFamily; want != got {
		t.Fatalf("unexpected error:\n-  want: %v\n-  got: %v", want, got)
	}
}

func TestLinuxConnReceive(t *testing.T) {
	// The request we sent netlink in the previous test; it will be echoed
	// back to us as part of this test
	req := Message{
		Header: Header{
			Length:   uint32(nlmsgAlign(nlmsgLength(4))),
			Flags:    HeaderFlagsRequest | HeaderFlagsAcknowledge,
			Sequence: 1,
			PID:      uint32(os.Getpid()),
		},
		Data: []byte{0x01, 0x02, 0x00, 0x00},
	}
	reqb, err := req.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal request to binary: %v", err)
	}

	res := Message{
		Header: Header{
			// 16 bytes: header
			//  4 bytes: error code
			// 20 bytes: request message
			Length:   uint32(nlmsgAlign(nlmsgLength(24))),
			Type:     HeaderTypeError,
			Sequence: 1,
			PID:      uint32(os.Getpid()),
		},
		// Error code "success", and copy of request
		Data: append([]byte{0x00, 0x00, 0x00, 0x00}, reqb...),
	}
	resb, err := res.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal response to binary: %v", err)
	}

	c, s := testLinuxConn(t, nil)

	from := &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
	}

	s.recvmsg.p = resb
	s.recvmsg.from = from

	msgs, err := c.Receive()
	if err != nil {
		t.Fatalf("failed to receive messages: %v", err)
	}

	if want, got := from, s.recvmsg.from; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected recvmsg address:\n- want: %#v\n-  got: %#v",
			want, got)
	}

	// Expect a MSG_PEEK and then no flags on second call
	if want, got := 2, len(s.recvmsg.flags); want != got {
		t.Fatalf("unexpected number of calls to recvmsg:\n- want: %v\n-  got: %v",
			want, got)
	}
	if want, got := unix.MSG_PEEK, s.recvmsg.flags[0]; want != got {
		t.Fatalf("unexpected first recvmsg flags:\n- want: %v\n-  got: %v",
			want, got)
	}
	if want, got := 0, s.recvmsg.flags[1]; want != got {
		t.Fatalf("unexpected second recvmsg flags:\n- want: %v\n-  got: %v",
			want, got)
	}

	if want, got := 1, len(msgs); want != got {
		t.Fatalf("unexpected number of messages:\n- want: %v\n-  got: %v",
			want, got)
	}

	if want, got := res, msgs[0]; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected output message:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestLinuxConnReceiveLargeMessage(t *testing.T) {
	n := os.Getpagesize() * 4

	res := Message{
		Header: Header{
			Length:   uint32(nlmsgAlign(nlmsgLength(n))),
			Type:     HeaderTypeError,
			Sequence: 1,
			PID:      uint32(os.Getpid()),
		},
		Data: make([]byte, n),
	}
	resb, err := res.MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal response to binary: %v", err)
	}

	c, s := testLinuxConn(t, nil)

	from := &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
	}

	s.recvmsg.p = resb
	s.recvmsg.from = from

	if _, err := c.Receive(); err != nil {
		t.Fatalf("failed to receive messages: %v", err)
	}

	// Expect several MSG_PEEK and then no flags
	want := []int{
		unix.MSG_PEEK,
		unix.MSG_PEEK,
		unix.MSG_PEEK,
		unix.MSG_PEEK,
		0,
	}

	if got := s.recvmsg.flags; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected number recvmsg flags:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestLinuxConnIntegration(t *testing.T) {
	const familyGeneric = 16

	c, err := Dial(familyGeneric, nil)
	if err != nil {
		t.Fatalf("failed to dial netlink: %v", err)
	}

	// Ask to send us an acknowledgement, which will contain an
	// error code (or success) and a copy of the payload we sent in
	req := Message{
		Header: Header{
			Flags: HeaderFlagsRequest | HeaderFlagsAcknowledge,
		},
	}

	// Perform a request, receive replies, and validate the replies
	msgs, err := c.Execute(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	if want, got := 1, len(msgs); want != got {
		t.Fatalf("unexpected message count from netlink:\n- want: %v\n-  got: %v",
			want, got)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("error closing netlink connection: %v", err)
	}

	m := msgs[0]

	if want, got := 0, int(nlenc.Uint32(m.Data[0:4])); want != got {
		t.Fatalf("unexpected error code:\n- want: %v\n-  got: %v", want, got)
	}

	if want, got := 36, int(m.Header.Length); want != got {
		t.Fatalf("unexpected header length:\n- want: %v\n-  got: %v", want, got)
	}
	if want, got := HeaderTypeError, m.Header.Type; want != got {
		t.Fatalf("unexpected header type:\n- want: %v\n-  got: %v", want, got)
	}
	if want, got := 0, int(m.Header.Flags); want != got {
		t.Fatalf("unexpected header flags:\n- want: %v\n-  got: %v", want, got)
	}

	// Sequence number not checked because we assign one at random when
	// a Conn is created.

	// Skip error code and unmarshal the copy of request sent back by
	// skipping the success code at bytes 0-4
	var reply Message
	if err := (&reply).UnmarshalBinary(m.Data[4:]); err != nil {
		t.Fatalf("failed to unmarshal reply: %v", err)
	}

	if want, got := req.Header.Flags, reply.Header.Flags; want != got {
		t.Fatalf("unexpected copy header flags:\n- want: %v\n-  got: %v", want, got)
	}
	if want, got := os.Getpid(), int(reply.Header.PID); want != got {
		t.Fatalf("unexpected copy header PID:\n- want: %v\n-  got: %v", want, got)
	}
	if want, got := len(req.Data), len(reply.Data); want != got {
		t.Fatalf("unexpected copy header data length:\n- want: %v\n-  got: %v", want, got)
	}
}

func TestLinuxConnIntegrationConcurrent(t *testing.T) {
	execN := func(n int, wg *sync.WaitGroup) {
		// It is important to lock this goroutine to its OS thread for the duration
		// of the netlink socket being used, or else the kernel may end up routing
		// messages to the wrong places.
		// See: http://lists.infradead.org/pipermail/libnl/2017-February/002293.html.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		const familyGeneric = 16

		c, err := Dial(familyGeneric, nil)
		if err != nil {
			panic(fmt.Sprintf("failed to dial netlink: %v", err))
		}

		req := Message{
			Header: Header{
				Flags: HeaderFlagsRequest | HeaderFlagsAcknowledge,
			},
		}

		for i := 0; i < n; i++ {
			vmsg, err := c.Send(req)
			if err != nil {
				if err == unix.EINVAL {
					// BUG(mdlayher): for reasons as of yet unknown, this Send will
					// occasionally return EINVAL.  For the time being, ignore this
					// error and keep the test itself running.  Needs more investigation.
					continue
				}

				panic(fmt.Sprintf("failed to send request: %v", err))
			}

			msgs, err := c.Receive()
			if err != nil {
				panic(fmt.Sprintf("failed to receive reply: %v", err))
			}

			if l := len(msgs); l != 1 {
				panic(fmt.Sprintf("unexpected number of reply messages: %d", l))
			}

			if err := Validate(vmsg, msgs); err != nil {
				panic(fmt.Sprintf("failed to validate request and reply: %v\n- req: %+v\n- rep: %+v",
					err, vmsg, msgs))
			}
		}

		_ = c.Close()
		wg.Done()
	}

	const (
		workers    = 16
		iterations = 10000
	)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go execN(iterations, &wg)
	}

	wg.Wait()
}

func TestLinuxConnJoinLeaveGroup(t *testing.T) {
	c, s := testLinuxConn(t, nil)

	group := uint32(1)

	if err := c.JoinGroup(group); err != nil {
		t.Fatalf("failed to join group: %v", err)
	}

	if err := c.LeaveGroup(group); err != nil {
		t.Fatalf("failed to leave group: %v", err)
	}

	l := uint32(unsafe.Sizeof(group))

	want := []setSockopt{
		{
			level: unix.SOL_NETLINK,
			name:  unix.NETLINK_ADD_MEMBERSHIP,
			v:     group,
			l:     l,
		},
		{
			level: unix.SOL_NETLINK,
			name:  unix.NETLINK_DROP_MEMBERSHIP,
			v:     group,
			l:     l,
		},
	}

	if got := s.setSockopt; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected socket options:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func TestLinuxConnConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		groups uint32
	}{
		{
			name:   "Default Config",
			config: &Config{},
			groups: 0x0,
		},
		{
			name:   "Config with Groups RTMGRP_IPV4_IFADDR",
			config: &Config{Groups: 0x10},
			groups: 0x10,
		},
		{
			name:   "Config with Groups RTMGRP_IPV4_IFADDR | RTMGRP_IPV4_ROUTE",
			config: &Config{Groups: 0x10 | 0x40},
			groups: 0x50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testLinuxConn(t, tt.config)

			if want, got := tt.groups, c.sa.Groups; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

func TestConnReceiveErrorLinux(t *testing.T) {
	// Note: using *Conn instead of Linux-only *conn, to test
	// error handling logic in *Conn.Receive

	tests := []struct {
		name string
		req  Message
		rep  [][]Message
		err  error
	}{
		{
			name: "ENOENT",
			rep: [][]Message{{{
				Header: Header{
					Length:   uint32(nlmsgAlign(nlmsgLength(4))),
					Type:     HeaderTypeError,
					Sequence: 1,
					PID:      1,
				},
				// -2, little endian (ENOENT)
				Data: []byte{0xfe, 0xff, 0xff, 0xff},
			}}},
			err: unix.ENOENT,
		},
		{
			name: "EINTR multipart",
			rep: [][]Message{
				{
					{
						Header: Header{
							Flags: HeaderFlagsMulti,
						},
					},
				},
				{
					{
						Header: Header{
							Type:  HeaderTypeError,
							Flags: HeaderFlagsMulti,
						},
						Data: []byte{0xfc, 0xff, 0xff, 0xff},
					},
				},
			},
			// -4, little endian (EINTR)
			err: unix.EINTR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, tc := testConn(t, 0)
			tc.receive = tt.rep

			_, err := c.Receive()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

func testLinuxConn(t *testing.T, config *Config) (*conn, *testSocket) {
	s := &testSocket{}
	c, _, err := bind(s, config)
	if err != nil {
		t.Fatalf("failed to bind: %v", err)
	}

	return c, s
}

type testSocket struct {
	bind           unix.Sockaddr
	bindErr        error
	closed         bool
	getsockname    unix.Sockaddr
	getsocknameErr error
	sendmsg        struct {
		p     []byte
		oob   []byte
		to    unix.Sockaddr
		flags int
	}
	recvmsg struct {
		// Received from caller
		flags []int
		// Sent to caller
		p         []byte
		oob       []byte
		recvflags int
		from      unix.Sockaddr
	}
	setSockopt []setSockopt
}

type setSockopt struct {
	level int
	name  int
	v     uint32
	l     uint32
}

func (s *testSocket) Bind(sa unix.Sockaddr) error {
	s.bind = sa
	return s.bindErr
}

func (s *testSocket) Close() error {
	s.closed = true
	return nil
}

func (s *testSocket) Getsockname() (unix.Sockaddr, error) {
	if s.getsockname == nil {
		return &unix.SockaddrNetlink{}, s.getsocknameErr
	}

	return s.getsockname, s.getsocknameErr
}

func (s *testSocket) Recvmsg(p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error) {
	s.recvmsg.flags = append(s.recvmsg.flags, flags)
	n := copy(p, s.recvmsg.p)
	oobn := copy(oob, s.recvmsg.oob)

	return n, oobn, s.recvmsg.recvflags, s.recvmsg.from, nil
}

func (s *testSocket) Sendmsg(p, oob []byte, to unix.Sockaddr, flags int) error {
	s.sendmsg.p = p
	s.sendmsg.oob = oob
	s.sendmsg.to = to
	s.sendmsg.flags = flags
	return nil
}

func (s *testSocket) SetSockopt(level, name int, v unsafe.Pointer, l uint32) error {
	s.setSockopt = append(s.setSockopt, setSockopt{
		level: level,
		name:  name,
		v:     *(*uint32)(v),
		l:     l,
	})

	return nil
}
