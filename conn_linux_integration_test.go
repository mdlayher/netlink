//+build integration linux

package netlink

import (
	"fmt"
	"os/exec"
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
		data, er := c.Receive()
		if er != nil {
			fmt.Printf("error in receive: %s\n", err)
		}
		in <- data
	}

	go recv()

	def := sudoIfCreate(t, "test0")
	defer def()

	timeout := time.After(5 * time.Second)
	var got []Message
	select {
	case got = <-in:
		break
	case <-timeout:
		t.Fatal("did not receive any messages after 5 seconds\n")
		break
	}
	t.Logf("received netlink data: %#v\n", got)
}

func sudoIfCreate(t *testing.T, ifName string) func() {
	cmd := exec.Command("sudo", "ip", "tuntap", "add", ifName, "mode", "tun")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("error creating tuntap device: %s\n", err)
	}
	cmd.Wait()

	return func() {
		cmd := exec.Command("sudo", "ip", "link", "del", ifName)
		err := cmd.Start()
		if err != nil {
			t.Fatalf("error removing tuntap device: %s\n", err)
		}
		cmd.Wait()
	}
}
