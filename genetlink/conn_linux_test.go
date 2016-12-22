//+build linux

package genetlink_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"
)

func TestLinuxConnIntegration(t *testing.T) {
	c, err := genetlink.Dial(nil)
	if err != nil {
		t.Fatalf("failed to dial generic netlink: %v", err)
	}

	// Ask netlink for TASKSTATS family; must be null-terminated string
	const name = "TASKSTATS"
	family := append([]byte(name), 0x00)

	const (
		commandGetFamily    = 3
		attributeFamilyName = 2
	)

	// Ask netlink for attributes about the specified family
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: commandGetFamily,
			Version: 1,
		},
		Data: func() []byte {
			ab, err := netlink.MarshalAttributes([]netlink.Attribute{{
				Type: attributeFamilyName,
				Data: family,
			}})
			if err != nil {
				t.Fatalf("failed to marshal attributes: %v", err)
			}

			return ab
		}(),
	}

	// Perform a request, receive replies, and validate the replies
	flags := netlink.HeaderFlagsRequest | netlink.HeaderFlagsAcknowledge
	msgs, err := c.Execute(req, flags)
	if err != nil {
		// TODO(mdlayher): this test currently does not work in Travis. See if
		// it's possible to get around that with some family
		if os.IsNotExist(err) {
			t.Skipf("skipping because %q family not available", name)
		}

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
	attrs, err := netlink.UnmarshalAttributes(m.Data)
	if err != nil {
		t.Fatalf("failed to unmarshal attributes: %v", err)
	}

	for _, a := range attrs {
		if a.Type != attributeFamilyName {
			continue
		}

		// Verify the family name we requested was returned to us
		if want, got := family, a.Data; !bytes.Equal(want, got) {
			t.Fatalf("unexpected family name:\n- want: [%# x]\n-  got: [%# x]", want, got)
		}
	}
}
