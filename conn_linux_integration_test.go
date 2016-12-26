//+build integration,linux

package netlink

import (
	"fmt"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

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
