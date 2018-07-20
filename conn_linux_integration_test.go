//+build integration,linux

package netlink

import (
	"fmt"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/bpf"
)

func TestLinuxNetlinkSetBPF(t *testing.T) {
	const familyGeneric = 16
	c, err := Dial(familyGeneric, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
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

	req := Message{
		Header: Header{
			Flags: HeaderFlagsRequest | HeaderFlagsAcknowledge,
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

func TestLinuxNetlinkSetBPFProgram(t *testing.T) {
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

func TestLinuxNetlinkMulticast(t *testing.T) {
	cfg := &Config{
		Groups: 0x1, // RTMGRP_LINK
	}

	c, err := Dial(0, cfg) // dials NETLINK_ROUTE
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	in := make(chan []Message)

	// routine for receiving any messages
	recv := func() {
		data, err := c.Receive()
		if err != nil {
			panic(fmt.Sprintf("error in receive: %s", err))
		}
		in <- data
	}

	go recv()

	ifName := "test0"

	def := sudoIfCreate(t, ifName)
	defer def()

	timeout := time.After(5 * time.Second)
	var data []Message
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

func sudoIfCreate(t *testing.T, ifName string) func() {
	var err error

	cmd := exec.Command("sudo", "ip", "tuntap", "add", ifName, "mode", "tun")
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

		cmd := exec.Command("sudo", "ip", "link", "del", ifName)
		err = cmd.Start()
		if err != nil {
			panic(fmt.Sprintf("error removing tuntap device: %s", err))
		}
		err = cmd.Wait()
		if err != nil {
			panic(fmt.Sprintf("error running command to remove tuntap device: %s", err))
		}
	}
}
