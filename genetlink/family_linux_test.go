//+build linux

package genetlink_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"
	"github.com/mdlayher/netlink/genetlink/genltest"
	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/sys/unix"
)

func TestConnGetFamily(t *testing.T) {
	const (
		name    = "nlctrl"
		version = 1
		flags   = netlink.HeaderFlagsRequest
	)

	wantgenl := genetlink.Message{
		Header: genetlink.Header{
			Command: unix.CTRL_CMD_GETFAMILY,
			Version: version,
		},
		Data: mustMarshalAttributes([]netlink.Attribute{{
			Type: unix.CTRL_ATTR_FAMILY_NAME,
			Data: nlenc.Bytes(name),
		}}),
	}

	c := genltest.Dial(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		if diff := cmp.Diff(flags, nreq.Header.Flags); diff != "" {
			t.Fatalf("unexpected netlink header flags (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(wantgenl, greq); diff != "" {
			t.Fatalf("unexpected generic netlink message (-want +got):\n%s", diff)
		}

		return []genetlink.Message{{
			Header: genetlink.Header{
				Command: unix.CTRL_CMD_NEWFAMILY,
				Version: version,
			},
			Data: mustMarshalAttributes([]netlink.Attribute{
				{
					Type: unix.CTRL_ATTR_FAMILY_NAME,
					Data: nlenc.Bytes(name),
				},
				{
					Type: unix.CTRL_ATTR_FAMILY_ID,
					Data: nlenc.Uint16Bytes(16),
				},
				{
					Type: unix.CTRL_ATTR_VERSION,
					Data: nlenc.Uint32Bytes(2),
				},
			}),
		}}, nil
	})

	family, err := c.GetFamily(name)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	wantFamily := genetlink.Family{
		ID:      16,
		Version: 2,
		Name:    name,
	}

	if diff := cmp.Diff(wantFamily, family); diff != "" {
		t.Fatalf("unexpected generic netlink family (-want +got):\n%s", diff)
	}
}

func TestConnFamilyList(t *testing.T) {
	const (
		version = 1
		flags   = netlink.HeaderFlagsRequest | netlink.HeaderFlagsDump
	)

	wantgenl := genetlink.Message{
		Header: genetlink.Header{
			Command: unix.CTRL_CMD_GETFAMILY,
			Version: version,
		},
		Data: []byte{},
	}

	c := genltest.Dial(func(greq genetlink.Message, nreq netlink.Message) ([]genetlink.Message, error) {
		if diff := cmp.Diff(flags, nreq.Header.Flags); diff != "" {
			t.Fatalf("unexpected netlink header flags (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(wantgenl, greq); diff != "" {
			t.Fatalf("unexpected generic netlink message (-want +got):\n%s", diff)
		}

		return []genetlink.Message{
			{
				Header: genetlink.Header{
					Command: unix.CTRL_CMD_NEWFAMILY,
					Version: version,
				},
				Data: mustMarshalAttributes([]netlink.Attribute{
					{
						Type: unix.CTRL_ATTR_FAMILY_NAME,
						Data: nlenc.Bytes("nlctrl"),
					},
					{
						Type: unix.CTRL_ATTR_FAMILY_ID,
						Data: nlenc.Uint16Bytes(16),
					},
					{
						Type: unix.CTRL_ATTR_VERSION,
						Data: nlenc.Uint32Bytes(2),
					},
				}),
			},
			{
				Header: genetlink.Header{
					Command: unix.CTRL_CMD_NEWFAMILY,
					Version: version,
				},
				Data: mustMarshalAttributes([]netlink.Attribute{
					{
						Type: unix.CTRL_ATTR_FAMILY_NAME,
						Data: nlenc.Bytes("nl80211"),
					},
					{
						Type: unix.CTRL_ATTR_FAMILY_ID,
						Data: nlenc.Uint16Bytes(26),
					},
					{
						Type: unix.CTRL_ATTR_VERSION,
						Data: nlenc.Uint32Bytes(1),
					},
				}),
				// Normally a "multi-part" done message would be here, but package
				// netlink takes care of trimming that away for us, so this package
				// assumes that has already been taken care of
			},
		}, nil
	})

	families, err := c.ListFamilies()
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	wantFamilies := []genetlink.Family{
		{
			ID:      16,
			Version: 2,
			Name:    "nlctrl",
		},
		{
			ID:      26,
			Version: 1,
			Name:    "nl80211",
		},
	}

	if diff := cmp.Diff(wantFamilies, families); diff != "" {
		t.Fatalf("unexpected generic netlink families (-want +got):\n%s", diff)
	}
}

func TestFamily_parseAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attrs []netlink.Attribute
		f     genetlink.Family
		ok    bool
	}{
		{
			name: "version too large",
			attrs: []netlink.Attribute{{
				Type: unix.CTRL_ATTR_VERSION,
				Data: []byte{0xff, 0x01, 0x00, 0x00},
			}},
		},
		{
			name: "bad multicast group array",
			attrs: []netlink.Attribute{{
				Type: unix.CTRL_ATTR_MCAST_GROUPS,
				Data: mustMarshalAttributes([]netlink.Attribute{
					{
						Type: 1,
					},
					{
						Type: 3,
					},
				}),
			}},
		},
		{
			name: "OK",
			attrs: []netlink.Attribute{
				{
					Type: unix.CTRL_ATTR_FAMILY_ID,
					Data: []byte{0x10, 0x00},
				},
				{
					Type: unix.CTRL_ATTR_FAMILY_NAME,
					Data: nlenc.Bytes("nlctrl"),
				},
				{
					Type: unix.CTRL_ATTR_VERSION,
					Data: []byte{0x02, 0x00, 0x00, 0x00},
				},
				{
					Type: unix.CTRL_ATTR_MCAST_GROUPS,
					Data: mustMarshalAttributes([]netlink.Attribute{
						{
							Type: 1,
							Data: mustMarshalAttributes([]netlink.Attribute{
								{
									Type: unix.CTRL_ATTR_MCAST_GRP_ID,
									Data: nlenc.Uint32Bytes(16),
								},
								{
									Type: unix.CTRL_ATTR_MCAST_GRP_NAME,
									Data: nlenc.Bytes("notify"),
								},
							}),
						},
						{
							Type: 2,
							Data: mustMarshalAttributes([]netlink.Attribute{
								{
									Type: unix.CTRL_ATTR_MCAST_GRP_ID,
									Data: nlenc.Uint32Bytes(17),
								},
								{
									Type: unix.CTRL_ATTR_MCAST_GRP_NAME,
									Data: nlenc.Bytes("foobar"),
								},
							}),
						},
					}),
				},
			},
			f: genetlink.Family{
				ID:      16,
				Version: 2,
				Name:    "nlctrl",
				Groups: []genetlink.MulticastGroup{
					{
						ID:   16,
						Name: "notify",
					},
					{
						ID:   17,
						Name: "foobar",
					},
				},
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := genltest.Dial(func(_ genetlink.Message, _ netlink.Message) ([]genetlink.Message, error) {
				return []genetlink.Message{{
					Data: mustMarshalAttributes(tt.attrs),
				}}, nil
			})

			family, err := c.GetFamily("")

			if err != nil && tt.ok {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && !tt.ok {
				t.Fatal("expected an error, but none occurred")
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tt.f, family); diff != "" {
				t.Fatalf("unexpected family (-want +got):\n%s", diff)
			}
		})
	}
}
