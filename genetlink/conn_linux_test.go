//+build linux

package genetlink_test

import (
	"net"
	"os"
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"
	"github.com/mdlayher/netlink/nlenc"
)

func TestLinuxConnFamilyGetIsNotExistIntegration(t *testing.T) {
	// Test that the documented behavior of returning an error that is compatible
	// with os.IsNotExist is correct
	const name = "NOTEXISTS"

	c, err := genetlink.Dial(nil)
	if err != nil {
		t.Fatalf("failed to dial generic netlink: %v", err)
	}

	if _, err := c.Family.Get(name); !os.IsNotExist(err) {
		t.Fatalf("expected not exists error, got: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("error closing netlink connection: %v", err)
	}
}

func TestLinuxConnFamilyGetIntegration(t *testing.T) {
	c, err := genetlink.Dial(nil)
	if err != nil {
		t.Fatalf("failed to dial generic netlink: %v", err)
	}

	const name = "nlctrl"
	family, err := c.Family.Get(name)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("skipping because %q family not available", name)
		}

		t.Fatalf("failed to query for family: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("error closing netlink connection: %v", err)
	}

	if want, got := name, family.Name; want != got {
		t.Fatalf("unexpected family name:\n- want: %q\n-  got: %q", want, got)
	}
}

func TestLinuxConnNL80211Integration(t *testing.T) {
	c, err := genetlink.Dial(nil)
	if err != nil {
		t.Fatalf("failed to dial generic netlink: %v", err)
	}

	const (
		name = "nl80211"

		nl80211CommandGetInterface = 5

		nl80211AttributeInterfaceIndex = 3
		nl80211AttributeInterfaceName  = 4
		nl80211AttributeAttributeMAC   = 6
	)

	family, err := c.Family.Get(name)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("skipping because %q family not available", name)
		}

		t.Fatalf("failed to query for family: %v", err)
	}

	req := genetlink.Message{
		Header: genetlink.Header{
			Command: nl80211CommandGetInterface,
			Version: family.Version,
		},
	}

	flags := netlink.HeaderFlagsRequest | netlink.HeaderFlagsDump
	msgs, err := c.Execute(req, family.ID, flags)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("error closing netlink connection: %v", err)
	}

	type ifInfo struct {
		Index int
		Name  string
		MAC   net.HardwareAddr
	}

	// Last message is end of multi-part indicator
	var infos []ifInfo
	for _, m := range msgs[:len(msgs)-1] {
		attrs, err := netlink.UnmarshalAttributes(m.Data)
		if err != nil {
			t.Fatalf("failed to unmarshal attributes: %v", err)
		}

		var info ifInfo
		for _, a := range attrs {
			switch a.Type {
			case nl80211AttributeInterfaceIndex:
				info.Index = int(nlenc.Uint32(a.Data))
			case nl80211AttributeInterfaceName:
				info.Name = nlenc.String(a.Data)
			case nl80211AttributeAttributeMAC:
				info.MAC = net.HardwareAddr(a.Data)
			}
		}

		infos = append(infos, info)
	}

	// Verify that nl80211 reported the same information as package net
	for _, info := range infos {
		// TODO(mdlayher): figure out why nl80211 returns a subdevice with
		// an empty name on newer kernel
		if info.Name == "" {
			continue
		}

		ifi, err := net.InterfaceByName(info.Name)
		if err != nil {
			t.Fatalf("error retrieving interface %q: %v", info.Name, err)
		}

		if want, got := ifi.Index, info.Index; want != got {
			t.Fatalf("unexpected interface index for %q:\n- want: %v\n-  got: %v",
				ifi.Name, want, got)
		}

		if want, got := ifi.Name, info.Name; want != got {
			t.Fatalf("unexpected interface name:\n- want: %q\n-  got: %q",
				want, got)
		}

		if want, got := ifi.HardwareAddr.String(), info.MAC.String(); want != got {
			t.Fatalf("unexpected interface MAC for %q:\n- want: %q\n-  got: %q",
				ifi.Name, want, got)
		}
	}
}

func TestLinuxConnFamilyListIntegration(t *testing.T) {
	c, err := genetlink.Dial(nil)
	if err != nil {
		t.Fatalf("failed to dial generic netlink: %v", err)
	}

	families, err := c.Family.List()
	if err != nil {
		t.Fatalf("failed to query for families: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("error closing netlink connection: %v", err)
	}

	// Should be at least nlctrl present
	var found bool
	const name = "nlctrl"
	for _, f := range families {
		if f.Name == name {
			found = true
		}
	}

	if !found {
		t.Fatalf("family %q was not found", name)
	}
}
