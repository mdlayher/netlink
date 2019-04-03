//+build integration,linux

package netlink_test

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
)

func TestLinuxConnIntegration(t *testing.T) {
	c, err := netlink.Dial(unix.NETLINK_GENERIC, nil)
	if err != nil {
		t.Fatalf("failed to dial netlink: %v", err)
	}

	// Ask to send us an acknowledgement, which will contain an
	// error code (or success) and a copy of the payload we sent in
	req := netlink.Message{
		Header: netlink.Header{
			Flags: netlink.Request | netlink.Acknowledge,
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
	if want, got := netlink.Error, m.Header.Type; want != got {
		t.Fatalf("unexpected header type:\n- want: %v\n-  got: %v", want, got)
	}
	// Recent kernel versions (> 4.14) return a 256 here instead of a 0
	if want, wantAlt, got := 0, 256, int(m.Header.Flags); want != got && wantAlt != got {
		t.Fatalf("unexpected header flags:\n- want: %v or %v\n-  got: %v", want, wantAlt, got)
	}

	// Sequence number not checked because we assign one at random when
	// a Conn is created.

	// Skip error code and unmarshal the copy of request sent back by
	// skipping the success code at bytes 0-4
	var reply netlink.Message
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

func TestLinuxConnIntegrationConcurrentRaceFree(t *testing.T) {
	c, err := netlink.Dial(unix.NETLINK_GENERIC, nil)
	if err != nil {
		t.Fatalf("failed to dial netlink: %v", err)
	}

	execN := func(n int) {
		req := netlink.Message{
			Header: netlink.Header{
				Flags: netlink.Request | netlink.Acknowledge,
			},
		}

		var res netlink.Message
		for i := 0; i < n; i++ {
			// Don't expect a "valid" request/reply because we are not serializing
			// our Send/Receive calls via Execute or with an external lock.
			//
			// Just verify that we don't trigger the race detector, we got a
			// valid netlink response, and it can be decoded as a valid
			// netlink message.
			if _, err := c.Send(req); err != nil {
				panicf("failed to send request: %v", err)
			}

			msgs, err := c.Receive()
			if err != nil {
				panicf("failed to receive reply: %v", err)
			}

			if l := len(msgs); l != 1 {
				panicf("unexpected number of reply messages: %d", l)
			}

			if err := res.UnmarshalBinary(msgs[0].Data[4:]); err != nil {
				panicf("failed to unmarshal reply: %v", err)
			}
		}
	}

	const (
		workers    = 16
		iterations = 10000
	)

	var wg sync.WaitGroup
	wg.Add(workers)
	defer wg.Wait()

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			execN(iterations)
		}()
	}
}

func TestLinuxConnIntegrationConcurrentReceiveClose(t *testing.T) {
	c, err := netlink.Dial(unix.NETLINK_GENERIC, nil)
	if err != nil {
		t.Fatalf("failed to dial netlink: %v", err)
	}

	// Verify this test cannot block indefinitely due to Receive hanging after
	// a call to Close.
	timer := time.AfterFunc(10*time.Second, func() {
		panic("test took too long")
	})
	defer timer.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()

	go func() {
		defer wg.Done()

		_, err := c.Receive()
		if err == nil {
			panicf("expected an error, but none occurred")
		}

		// Expect an error due to file descriptor being closed.
		serr := err.(*netlink.OpError).Err.(*os.SyscallError).Err
		if diff := cmp.Diff(unix.EBADF, serr); diff != "" {
			t.Fatalf("unexpected error from receive (-want +got):\n%s", diff)
		}
	}()

	if err := c.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}
}

func TestLinuxConnIntegrationConcurrentSerializeExecute(t *testing.T) {
	c, err := netlink.Dial(unix.NETLINK_GENERIC, nil)
	if err != nil {
		t.Fatalf("failed to dial netlink: %v", err)
	}

	execN := func(n int) {
		req := netlink.Message{
			Header: netlink.Header{
				Flags: netlink.Request | netlink.Acknowledge,
			},
		}

		for i := 0; i < n; i++ {
			// Execute will internally call Validate to ensure its
			// request/response transaction is serialized appropriately, and
			// any errors doing so will be reported here.
			if _, err := c.Execute(req); err != nil {
				panicf("failed to execute: %v", err)
			}
		}
	}

	const (
		workers    = 4
		iterations = 2000
	)

	var wg sync.WaitGroup
	wg.Add(workers)
	defer wg.Wait()

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			execN(iterations)
		}()
	}
}

func TestLinuxConnIntegrationClosedConn(t *testing.T) {
	c, err := netlink.Dial(unix.NETLINK_GENERIC, nil)
	if err != nil {
		t.Fatalf("failed to dial netlink: %v", err)
	}

	// Close the connection immediately and ensure that future calls get EBADF.
	if err := c.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	_, err = c.Receive()

	serr := err.(*netlink.OpError).Err.(*os.SyscallError).Err
	if diff := cmp.Diff(unix.EBADF, serr); diff != "" {
		t.Fatalf("unexpected error from receive (-want +got):\n%s", diff)
	}
}

func TestLinuxConnIntegrationSetBuffersSyscallConn(t *testing.T) {
	c, err := netlink.Dial(unix.NETLINK_GENERIC, nil)
	if err != nil {
		t.Fatalf("failed to dial netlink: %v", err)
	}
	defer c.Close()

	const (
		set = 8192

		// Per man 7 socket:
		//
		// "The kernel doubles this value (to allow space for book‐keeping
		// overhead) when it is set using setsockopt(2), and this doubled value
		// is returned by getsockopt(2).""
		want = set * 2
	)

	if err := c.SetReadBuffer(set); err != nil {
		t.Fatalf("failed to set read buffer size: %v", err)
	}

	if err := c.SetWriteBuffer(set); err != nil {
		t.Fatalf("failed to set write buffer size: %v", err)
	}

	// Now that we've set the buffers, we can check the size by asking the
	// kernel using SyscallConn and getsockopt.

	rc, err := c.SyscallConn()
	if err != nil {
		t.Fatalf("failed to get syscall conn: %v", err)
	}

	mustSize := func(opt int) int {
		var (
			value int
			serr  error
		)

		err := rc.Control(func(fd uintptr) {
			value, serr = unix.GetsockoptInt(int(fd), unix.SOL_SOCKET, opt)
		})
		if err != nil {
			t.Fatalf("failed to call control: %v", err)
		}
		if serr != nil {
			t.Fatalf("failed to call getsockopt: %v", serr)
		}

		return value
	}

	if diff := cmp.Diff(want, mustSize(unix.SO_RCVBUF)); diff != "" {
		t.Fatalf("unexpected read buffer size (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, mustSize(unix.SO_SNDBUF)); diff != "" {
		t.Fatalf("unexpected write buffer size (-want +got):\n%s", diff)
	}
}

func TestLinuxConnIntegrationSetBPF(t *testing.T) {
	c, err := netlink.Dial(unix.NETLINK_GENERIC, nil)
	if err != nil {
		t.Fatalf("failed to dial netlink: %v", err)
	}
	defer c.Close()

	// The sequence number which will be permitted by the BPF filter.
	// Using max uint32 helps us avoid dealing with host (netlink) vs
	// network (BPF) endianness during this test.
	const sequence uint32 = 0xffffffff

	prog, err := bpf.Assemble(testBPFProgram(sequence))
	if err != nil {
		t.Fatalf("failed to assemble BPF program: %v", err)
	}

	if err := c.SetBPF(prog); err != nil {
		t.Fatalf("failed to attach BPF program to socket: %v", err)
	}

	req := netlink.Message{
		Header: netlink.Header{
			Flags: netlink.Request | netlink.Acknowledge,
		},
	}

	sequences := []struct {
		seq uint32
		ok  bool
	}{
		// OK, bad, OK.  Expect two messages to be received.
		{seq: sequence, ok: true},
		{seq: 10, ok: false},
		{seq: sequence, ok: true},
	}

	for _, s := range sequences {
		req.Header.Sequence = s.seq
		if _, err := c.Send(req); err != nil {
			t.Fatalf("failed to send with sequence %d: %v", s.seq, err)
		}

		if !s.ok {
			continue
		}

		msgs, err := c.Receive()
		if err != nil {
			t.Fatalf("failed to receive with sequence %d: %v", s.seq, err)
		}

		// Make sure the received message has the expected sequence number.
		if l := len(msgs); l != 1 {
			t.Fatalf("unexpected number of messages: %d", l)
		}

		if want, got := s.seq, msgs[0].Header.Sequence; want != got {
			t.Fatalf("unexpected reply sequence number:\n- want: %v\n-  got: %v",
				want, got)
		}
	}
	if err := c.RemoveBPF(); err != nil {
		t.Fatalf("failed to remove BPF filter: %v", err)
	}
}

func TestLinuxConnIntegration_testBPFProgram(t *testing.T) {
	vm, err := bpf.NewVM(testBPFProgram(0xffffffff))
	if err != nil {
		t.Fatalf("failed to create BPF VM: %v", err)
	}

	msg := []byte{
		0x10, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x01, 0x00,
		// Allowed sequence number.
		0xff, 0xff, 0xff, 0xff,
		0x01, 0x00, 0x00, 0x00,
	}

	out, err := vm.Run(msg)
	if err != nil {
		t.Fatalf("failed to execute OK input: %v", err)
	}
	if out == 0 {
		t.Fatal("BPF filter dropped OK input")
	}

	msg = []byte{
		0x10, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x01, 0x00,
		// Bad sequence number.
		0x00, 0x11, 0x22, 0x33,
		0x01, 0x00, 0x00, 0x00,
	}

	out, err = vm.Run(msg)
	if err != nil {
		t.Fatalf("failed to execute bad input: %v", err)
	}
	if out != 0 {
		t.Fatal("BPF filter did not drop bad input")
	}
}

// testBPFProgram returns a BPF program which only allows frames with the
// input sequence number.
func testBPFProgram(allowSequence uint32) []bpf.Instruction {
	return []bpf.Instruction{
		bpf.LoadAbsolute{
			Off:  8,
			Size: 4,
		},
		bpf.JumpIf{
			Cond:     bpf.JumpEqual,
			Val:      allowSequence,
			SkipTrue: 1,
		},
		bpf.RetConstant{
			Val: 0,
		},
		bpf.RetConstant{
			Val: 128,
		},
	}
}

func TestLinuxConnIntegrationMulticast(t *testing.T) {
	c, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{
		Groups: 0x1, // RTMGRP_LINK
	})
	if err != nil {
		t.Fatalf("failed to dial netlink: %v", err)
	}

	in := make(chan []netlink.Message)

	// routine for receiving any messages
	recv := func() {
		data, err := c.Receive()
		if err != nil {
			panicf("error in receive: %s", err)
		}
		in <- data
	}

	go recv()

	ifName := "test0"

	def := createInterface(t, ifName)
	defer def()

	timeout := time.After(5 * time.Second)
	var data []netlink.Message
	select {
	case data = <-in:
		break
	case <-timeout:
		panic("did not receive any messages after 5 seconds")
	}

	interf := []byte(ifName)
	want := make([]uint8, len(ifName))
	copy(want, interf[:])

	got := make([]uint8, len(ifName))
	copy(got, data[0].Data[20:len(ifName)+20])

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("received message does not mention ifName %q", ifName)
	}
}

func createInterface(t *testing.T, ifName string) func() {
	var err error

	cmd := exec.Command("ip", "tuntap", "add", ifName, "mode", "tun")
	err = cmd.Start()
	if err != nil {
		t.Fatalf("error creating tuntap device: %s", err)
		return func() {}
	}
	err = cmd.Wait()
	if err != nil {
		t.Fatalf("error running command to create tuntap device: %s", err)
		return func() {}
	}

	return func() {
		var err error

		cmd := exec.Command("ip", "link", "del", ifName)
		err = cmd.Start()
		if err != nil {
			panicf("error removing tuntap device: %s", err)
		}
		err = cmd.Wait()
		if err != nil {
			panicf("error running command to remove tuntap device: %s", err)
		}
	}
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
