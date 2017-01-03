package genetlink

import (
	"reflect"
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
)

func TestConnFamilyGet(t *testing.T) {
	const name = "nlctrl"

	wantnl := netlink.Message{
		Header: netlink.Header{
			Type:  Protocol,
			Flags: netlink.HeaderFlagsRequest,
		},
		Data: mustMarshal(Message{
			Header: Header{
				Command: ctrlCommandGetFamily,
				Version: ctrlVersion,
			},
			Data: mustMarshalAttributes([]netlink.Attribute{{
				Type: attrFamilyName,
				Data: nlenc.Bytes(name),
			}}),
		}),
	}

	c, tc := testConn(t)
	tc.receive = []netlink.Message{{
		Header: netlink.Header{
			Length: headerLen + 12 + 8 + 8,
		},
		Data: mustMarshal(Message{
			Header: Header{
				Command: 0x1,
				Version: 0x2,
			},
			Data: mustMarshalAttributes([]netlink.Attribute{
				{
					Length: 11,
					Type:   attrFamilyName,
					Data:   nlenc.Bytes(name),
				},
				{
					Length: 6,
					Type:   attrFamilyID,
					Data:   nlenc.Uint16Bytes(16),
				},
				{
					Length: 8,
					Type:   attrVersion,
					Data:   nlenc.Uint32Bytes(2),
				},
			}),
		}),
	}}

	family, err := c.Family.Get(name)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if want, got := wantnl, tc.send; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected request:\n- want: %#v\n-  got: %#v",
			want, got)
	}

	want := Family{
		ID:      16,
		Version: 2,
		Name:    name,
	}

	if got := family; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected family:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestConnFamilyList(t *testing.T) {
	wantnl := netlink.Message{
		Header: netlink.Header{
			Type:  Protocol,
			Flags: netlink.HeaderFlagsRequest | netlink.HeaderFlagsDump,
		},
		Data: mustMarshal(Message{
			Header: Header{
				Command: ctrlCommandGetFamily,
				Version: ctrlVersion,
			},
		}),
	}

	c, tc := testConn(t)
	tc.receive = []netlink.Message{
		{
			Header: netlink.Header{
				Length: headerLen + 12 + 8 + 8,
				Flags:  netlink.HeaderFlagsMulti,
			},
			Data: mustMarshal(Message{
				Header: Header{
					Command: 0x1,
					Version: 0x2,
				},
				Data: mustMarshalAttributes([]netlink.Attribute{
					{
						Length: 11,
						Type:   attrFamilyName,
						Data:   nlenc.Bytes("nlctrl"),
					},
					{
						Length: 6,
						Type:   attrFamilyID,
						Data:   nlenc.Uint16Bytes(16),
					},
					{
						Length: 8,
						Type:   attrVersion,
						Data:   nlenc.Uint32Bytes(2),
					},
				}),
			}),
		},
		{
			Header: netlink.Header{
				Length: headerLen + 12 + 8 + 8,
				Flags:  netlink.HeaderFlagsMulti,
			},
			Data: mustMarshal(Message{
				Header: Header{
					Command: 0x1,
					Version: 0x2,
				},
				Data: mustMarshalAttributes([]netlink.Attribute{
					{
						Length: 12,
						Type:   attrFamilyName,
						Data:   nlenc.Bytes("nl80211"),
					},
					{
						Length: 6,
						Type:   attrFamilyID,
						Data:   nlenc.Uint16Bytes(26),
					},
					{
						Length: 8,
						Type:   attrVersion,
						Data:   nlenc.Uint32Bytes(1),
					},
				}),
			}),
		},
		// Normally a "multi-part" done message would be here, but package
		// netlink takes care of trimming that away for us, so this package
		// assumes that has already been taken care of
	}

	families, err := c.Family.List()
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	if want, got := wantnl, tc.send; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected request:\n- want: %#v\n-  got: %#v",
			want, got)
	}

	want := []Family{
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

	if got := families; !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected families:\n- want: %#v\n-  got: %#v",
			want, got)
	}
}

func TestFamily_parseAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attrs []netlink.Attribute
		f     *Family
		err   error
	}{
		{
			name: "version too large",
			attrs: []netlink.Attribute{{
				Type: attrVersion,
				Data: []byte{0xff, 0x01, 0x00, 0x00},
			}},
			err: errInvalidFamilyVersion,
		},
		{
			name: "bad multicast group array",
			attrs: []netlink.Attribute{{
				Type: attrMulticastGroups,
				Data: mustMarshalAttributes([]netlink.Attribute{
					{
						Type: 1,
					},
					{
						Type: 3,
					},
				}),
			}},
			err: errInvalidMulticastGroupArray,
		},
		{
			name: "OK",
			attrs: []netlink.Attribute{
				{
					Type: attrFamilyID,
					Data: []byte{0x10, 0x00},
				},
				{
					Type: attrFamilyName,
					Data: nlenc.Bytes("nlctrl"),
				},
				{
					Type: attrVersion,
					Data: []byte{0x02, 0x00, 0x00, 0x00},
				},
				{
					Type: attrMulticastGroups,
					Data: mustMarshalAttributes([]netlink.Attribute{
						{
							Type: 1,
							Data: mustMarshalAttributes([]netlink.Attribute{
								{
									Type: attrMGID,
									Data: nlenc.Uint32Bytes(16),
								},
								{
									Type: attrMGName,
									Data: nlenc.Bytes("notify"),
								},
							}),
						},
						{
							Type: 2,
							Data: mustMarshalAttributes([]netlink.Attribute{
								{
									Type: attrMGID,
									Data: nlenc.Uint32Bytes(17),
								},
								{
									Type: attrMGName,
									Data: nlenc.Bytes("foobar"),
								},
							}),
						},
					}),
				},
			},
			f: &Family{
				ID:      16,
				Version: 2,
				Name:    "nlctrl",
				Groups: []MulticastGroup{
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f Family
			err := (&f).parseAttributes(tt.attrs)

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %#v\n-  got: %#v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := *tt.f, f; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Family:\n- want: %#v\n-  got: %#v",
					want, got)
			}
		})
	}
}
