package netlink

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
	"unsafe"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/netlink/nlenc"
)

func TestMarshalAttributes(t *testing.T) {
	skipBigEndian(t)

	tests := []struct {
		name  string
		attrs []Attribute
		b     []byte
		err   error
	}{
		{
			name: "one attribute, short length",
			attrs: []Attribute{{
				Length: 3,
				Type:   1,
			}},
			err: errInvalidAttribute,
		},
		{
			name: "one attribute, no data",
			attrs: []Attribute{{
				Length: 4,
				Type:   1,
				Data:   make([]byte, 0),
			}},
			b: []byte{
				0x04, 0x00,
				0x01, 0x00,
			},
		},
		{
			name: "one attribute, no data, length calculated",
			attrs: []Attribute{{
				Type: 1,
				Data: make([]byte, 0),
			}},
			b: []byte{
				0x04, 0x00,
				0x01, 0x00,
			},
		},
		{
			name: "one attribute, padded",
			attrs: []Attribute{{
				Length: 5,
				Type:   1,
				Data:   []byte{0xff},
			}},
			b: []byte{
				0x05, 0x00,
				0x01, 0x00,
				0xff, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "one attribute, padded, length calculated",
			attrs: []Attribute{{
				Type: 1,
				Data: []byte{0xff},
			}},
			b: []byte{
				0x05, 0x00,
				0x01, 0x00,
				0xff, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "one attribute, aligned",
			attrs: []Attribute{{
				Length: 8,
				Type:   2,
				Data:   []byte{0xaa, 0xbb, 0xcc, 0xdd},
			}},
			b: []byte{
				0x08, 0x00,
				0x02, 0x00,
				0xaa, 0xbb, 0xcc, 0xdd,
			},
		},
		{
			name: "one attribute, aligned, length calculated",
			attrs: []Attribute{{
				Type: 2,
				Data: []byte{0xaa, 0xbb, 0xcc, 0xdd},
			}},
			b: []byte{
				0x08, 0x00,
				0x02, 0x00,
				0xaa, 0xbb, 0xcc, 0xdd,
			},
		},
		{
			name: "multiple attributes",
			attrs: []Attribute{
				{
					Length: 5,
					Type:   1,
					Data:   []byte{0xff},
				},
				{
					Length: 8,
					Type:   2,
					Data:   []byte{0xaa, 0xbb, 0xcc, 0xdd},
				},
				{
					Length: 4,
					Type:   3,
					Data:   make([]byte, 0),
				},
				{
					Length: 16,
					Type:   4,
					Data: []byte{
						0x11, 0x11, 0x11, 0x11,
						0x22, 0x22, 0x22, 0x22,
						0x33, 0x33, 0x33, 0x33,
					},
				},
			},
			b: []byte{
				// 1
				0x05, 0x00,
				0x01, 0x00,
				0xff, 0x00, 0x00, 0x00,
				// 2
				0x08, 0x00,
				0x02, 0x00,
				0xaa, 0xbb, 0xcc, 0xdd,
				// 3
				0x04, 0x00,
				0x03, 0x00,
				// 4
				0x10, 0x00,
				0x04, 0x00,
				0x11, 0x11, 0x11, 0x11,
				0x22, 0x22, 0x22, 0x22,
				0x33, 0x33, 0x33, 0x33,
			},
		},
		{
			name: "multiple attributes, length calculated",
			attrs: []Attribute{
				{
					Type: 1,
					Data: []byte{0xff},
				},
				{
					Type: 2,
					Data: []byte{0xaa, 0xbb, 0xcc, 0xdd},
				},
				{
					Type: 3,
					Data: make([]byte, 0),
				},
				{
					Type: 4,
					Data: []byte{
						0x11, 0x11, 0x11, 0x11,
						0x22, 0x22, 0x22, 0x22,
						0x33, 0x33, 0x33, 0x33,
					},
				},
			},
			b: []byte{
				// 1
				0x05, 0x00,
				0x01, 0x00,
				0xff, 0x00, 0x00, 0x00,
				// 2
				0x08, 0x00,
				0x02, 0x00,
				0xaa, 0xbb, 0xcc, 0xdd,
				// 3
				0x04, 0x00,
				0x03, 0x00,
				// 4
				0x10, 0x00,
				0x04, 0x00,
				0x11, 0x11, 0x11, 0x11,
				0x22, 0x22, 0x22, 0x22,
				0x33, 0x33, 0x33, 0x33,
			},
		},
		{
			name: "max type space, length 0",
			attrs: []Attribute{
				{
					Length: 4,
					Type:   0xffff,
					Data:   make([]byte, 0),
				},
			},
			b: []byte{
				0x04, 0x00,
				0xff, 0xff,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := MarshalAttributes(tt.attrs)

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.b, b; !bytes.Equal(want, got) {
				t.Fatalf("unexpected bytes:\n- want: [%# x]\n-  got: [%# x]",
					want, got)
			}
		})
	}
}

func TestUnmarshalAttributes(t *testing.T) {
	skipBigEndian(t)

	tests := []struct {
		name  string
		b     []byte
		attrs []Attribute
		err   error
	}{
		{
			name: "empty slice",
		},
		{
			name: "short slice",
			b:    make([]byte, 3),
			err:  errInvalidAttribute,
		},
		{
			name: "length too short (<4 bytes)",
			b: []byte{
				0x03, 0x00,
				0x00,
			},
			err: errInvalidAttribute,
		},
		{
			name: "length too long",
			b: []byte{
				0xff, 0xff,
				0x00, 0x00,
			},
			err: errInvalidAttribute,
		},
		{
			name: "one attribute, not aligned",
			b: []byte{
				0x05, 0x00,
				0x01, 0x00,
				0xff,
			},
			err: errInvalidAttribute,
		},
		{
			name: "fuzz crasher: length 1, too short",
			b:    []byte("\x01\x0000"),
			err:  errInvalidAttribute,
		},
		{
			name: "no attributes, length 0",
			b: []byte{
				0x00, 0x00,
				0x00, 0x00,
			},
		},
		{
			name: "one attribute, no data",
			b: []byte{
				0x04, 0x00,
				0x01, 0x00,
			},
			attrs: []Attribute{{
				Length: 4,
				Type:   1,
				Data:   make([]byte, 0),
			}},
		},
		{
			name: "one attribute, padded",
			b: []byte{
				0x05, 0x00,
				0x01, 0x00,
				0xff, 0x00, 0x00, 0x00,
			},
			attrs: []Attribute{{
				Length: 5,
				Type:   1,
				Data:   []byte{0xff},
			}},
		},
		{
			name: "one attribute, aligned",
			b: []byte{
				0x08, 0x00,
				0x02, 0x00,
				0xaa, 0xbb, 0xcc, 0xdd,
			},
			attrs: []Attribute{{
				Length: 8,
				Type:   2,
				Data:   []byte{0xaa, 0xbb, 0xcc, 0xdd},
			}},
		},
		{
			name: "multiple attributes",
			b: []byte{
				// 1
				0x05, 0x00,
				0x01, 0x00,
				0xff, 0x00, 0x00, 0x00,
				// 2
				0x08, 0x00,
				0x02, 0x00,
				0xaa, 0xbb, 0xcc, 0xdd,
				// 3
				0x04, 0x00,
				0x03, 0x00,
				// 4
				0x10, 0x00,
				0x04, 0x00,
				0x11, 0x11, 0x11, 0x11,
				0x22, 0x22, 0x22, 0x22,
				0x33, 0x33, 0x33, 0x33,
			},
			attrs: []Attribute{
				{
					Length: 5,
					Type:   1,
					Data:   []byte{0xff},
				},
				{
					Length: 8,
					Type:   2,
					Data:   []byte{0xaa, 0xbb, 0xcc, 0xdd},
				},
				{
					Length: 4,
					Type:   3,
					Data:   make([]byte, 0),
				},
				{
					Length: 16,
					Type:   4,
					Data: []byte{
						0x11, 0x11, 0x11, 0x11,
						0x22, 0x22, 0x22, 0x22,
						0x33, 0x33, 0x33, 0x33,
					},
				},
			},
		},
		{
			name: "max type space, length 0",
			b: []byte{
				0x04, 0x00,
				0xff, 0xff,
			},
			attrs: []Attribute{
				{
					Length: 4,
					Type:   0xffff,
					Data:   make([]byte, 0),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs, err := UnmarshalAttributes(tt.b)

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v",
					want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.attrs, attrs; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected attributes:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

func TestAttributeDecoderError(t *testing.T) {
	bad := []Attribute{{
		Type: 1,
		// Doesn't fit any integer types.
		Data: []byte{0xe, 0xad, 0xbe},
	}}

	skipBigEndian(t)

	tests := []struct {
		name  string
		attrs []Attribute
		fn    func(ad *AttributeDecoder)
	}{
		{
			name:  "uint8",
			attrs: bad,
			fn: func(ad *AttributeDecoder) {
				ad.Uint8()
				ad.Next()
				ad.Uint8()
			},
		},
		{
			name:  "uint16",
			attrs: bad,
			fn: func(ad *AttributeDecoder) {
				ad.Uint16()
				ad.Next()
				ad.Uint16()
			},
		},
		{
			name:  "uint32",
			attrs: bad,
			fn: func(ad *AttributeDecoder) {
				ad.Uint32()
				ad.Next()
				ad.Uint32()
			},
		},
		{
			name:  "uint64",
			attrs: bad,
			fn: func(ad *AttributeDecoder) {
				ad.Uint64()
				ad.Next()
				ad.Uint64()
			},
		},
		{
			name:  "do",
			attrs: bad,
			fn: func(ad *AttributeDecoder) {
				ad.Do(func(_ []byte) error {
					return errors.New("some error")
				})
				ad.Do(func(_ []byte) error {
					panic("shouldn't be called")
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := MarshalAttributes(tt.attrs)
			if err != nil {
				t.Fatalf("failed to marshal attributes: %v", err)
			}

			ad, err := NewAttributeDecoder(b)
			if err != nil {
				t.Fatalf("failed to create attribute decoder: %v", err)
			}

			for ad.Next() {
				tt.fn(ad)
			}

			if err := ad.Err(); err == nil {
				t.Fatal("expected an error, but none occurred")
			}
		})
	}
}

func TestAttributeDecoderOK(t *testing.T) {
	skipBigEndian(t)

	tests := []struct {
		name   string
		attrs  []Attribute
		endian binary.ByteOrder
		fn     func(ad *AttributeDecoder)
	}{
		{
			name:  "empty",
			attrs: nil,
			fn: func(_ *AttributeDecoder) {
				panic("should not be called")
			},
		},
		{
			name:  "uint native endian",
			attrs: adEndianAttrs(nlenc.NativeEndian()),
			fn:    adEndianTest(nlenc.NativeEndian()),
		},
		{
			name:  "uint little endian",
			attrs: adEndianAttrs(binary.LittleEndian),
			fn:    adEndianTest(binary.LittleEndian),
		},
		{
			name:  "uint big endian",
			attrs: adEndianAttrs(binary.BigEndian),
			fn:    adEndianTest(binary.BigEndian),
		},
		{
			name: "string",
			attrs: []Attribute{{
				Type: 1,
				Data: nlenc.Bytes("hello world"),
			}},
			fn: func(ad *AttributeDecoder) {
				var s string
				switch t := ad.Type(); t {
				case 1:
					s = ad.String()
				default:
					panicf("unhandled attribute type: %d", t)
				}

				if diff := cmp.Diff("hello world", s); diff != "" {
					panicf("unexpected attribute value (-want +got):\n%s", diff)
				}
			},
		},
		{
			name: "do",
			attrs: []Attribute{
				// Arbitrary C-like structure.
				{
					Type: 1,
					Data: []byte{0xde, 0xad, 0xbe},
				},
				// Nested attributes.
				{
					Type: 2,
					Data: func() []byte {
						b, err := MarshalAttributes([]Attribute{{
							Type: 2,
							Data: nlenc.Uint16Bytes(2),
						}})
						if err != nil {
							panicf("failed to marshal test attributes: %v", err)
						}

						return b
					}(),
				},
			},
			fn: func(ad *AttributeDecoder) {
				switch t := ad.Type(); t {
				case 1:
					type cstruct struct {
						A uint16
						B uint8
					}

					want := cstruct{
						// Little-endian is the worst.
						A: 0xadde,
						B: 0xbe,
					}

					ad.Do(func(b []byte) error {
						got := *(*cstruct)(unsafe.Pointer(&b[0]))

						if diff := cmp.Diff(want, got); diff != "" {
							panicf("unexpected struct (-want +got):\n%s", diff)
						}

						return nil
					})
				case 2:
					ad.Do(func(b []byte) error {
						adi, err := NewAttributeDecoder(b)
						if err != nil {
							return err
						}

						var got int
						first := true
						for adi.Next() {
							if !first {
								panic("loop iterated too many times")
							}
							first = false

							if adi.Type() != 2 {
								panicf("unhandled attribute type: %d", t)
							}

							got = int(adi.Uint16())
						}

						if diff := cmp.Diff(2, got); diff != "" {
							panicf("unexpected nested attribute value (-want +got):\n%s", diff)
						}

						return adi.Err()
					})
				default:
					panicf("unhandled attribute type: %d", t)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := MarshalAttributes(tt.attrs)
			if err != nil {
				t.Fatalf("failed to marshal attributes: %v", err)
			}

			ad, err := NewAttributeDecoder(b)
			if err != nil {
				t.Fatalf("failed to create attribute decoder: %v", err)
			}

			for ad.Next() {
				tt.fn(ad)
			}

			if err := ad.Err(); err != nil {
				t.Fatalf("failed to decode attributes: %v", err)
			}
		})
	}
}

func adEndianAttrs(order binary.ByteOrder) []Attribute {
	return []Attribute{
		{
			Type: 1,
			Data: func() []byte {
				return []byte{1}
			}(),
		},
		{
			Type: 2,
			Data: func() []byte {
				b := make([]byte, 2)
				order.PutUint16(b, 2)
				return b
			}(),
		},
		{
			Type: 3,
			Data: func() []byte {
				b := make([]byte, 4)
				order.PutUint32(b, 3)
				return b
			}(),
		},
		{
			Type: 4,
			Data: func() []byte {
				b := make([]byte, 8)
				order.PutUint64(b, 4)
				return b
			}(),
		},
	}
}

func adEndianTest(order binary.ByteOrder) func(ad *AttributeDecoder) {
	return func(ad *AttributeDecoder) {
		ad.ByteOrder = order

		var (
			t uint16
			v int
		)

		switch t = ad.Type(); t {
		case 1:
			v = int(ad.Uint8())
		case 2:
			v = int(ad.Uint16())
		case 3:
			v = int(ad.Uint32())
		case 4:
			v = int(ad.Uint64())
		default:
			panicf("unhandled attribute type: %d", t)
		}

		if diff := cmp.Diff(int(t), v); diff != "" {
			panicf("unexpected attribute value (-want +got):\n%s", diff)
		}
	}
}
