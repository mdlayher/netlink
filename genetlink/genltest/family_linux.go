//+build linux

package genltest

import (
	"fmt"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"
	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/sys/unix"
)

// serveFamily is the Linux implementation of ServeFamily.
func serveFamily(f genetlink.Family, fn Func) Func {
	return func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		// Only intercept "get family" commands to the generic netlink controller.
		if nreq.Header.Type != unix.GENL_ID_CTRL || greq.Header.Command != unix.CTRL_CMD_GETFAMILY {
			return fn(greq, nreq)
		}

		attrs, err := netlink.UnmarshalAttributes(greq.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse get family request attributes: %v", err)
		}

		// Ensure this request is for the family provided by f.
		for _, a := range attrs {
			if want, got := unix.CTRL_ATTR_FAMILY_NAME, int(a.Type); want != got {
				return nil, fmt.Errorf("unexpected get family request attribute: %d, want: %d", got, want)
			}

			if want, got := f.Name, nlenc.String(a.Data); want != got {
				return nil, fmt.Errorf("unexpected get family request value: %q, want: %q", got, want)
			}
		}

		// Return the family information for f.
		// TODO(mdlayher): return multicast groups and other attributes.
		attrb, err := netlink.MarshalAttributes([]netlink.Attribute{
			{
				Type: unix.CTRL_ATTR_FAMILY_ID,
				Data: nlenc.Uint16Bytes(f.ID),
			},
			{
				Type: unix.CTRL_ATTR_FAMILY_NAME,
				Data: nlenc.Bytes(f.Name),
			},
			{
				Type: unix.CTRL_ATTR_VERSION,
				Data: nlenc.Uint32Bytes(uint32(f.Version)),
			},
		})
		if err != nil {
			return nil, err
		}

		return []genetlink.Message{{
			Header: genetlink.Header{
				Command: unix.CTRL_CMD_NEWFAMILY,
				// TODO(mdlayher): constant nlctrl version number?
				Version: 2,
			},
			Data: attrb,
		}}, nil
	}
}
