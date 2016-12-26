//+build linux

package genetlink_test

import (
	"os"
	"testing"

	"github.com/mdlayher/netlink/genetlink"
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
