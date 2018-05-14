package netlink

import (
	"bytes"
	"reflect"
	"testing"
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
			name: "nested and endianness bits",
			attrs: []Attribute{
				{
					Length:       4,
					Type:         0,
					Nested:       true,
					NetByteOrder: true,
					Data:         make([]byte, 0),
				},
			},
			err: errInvalidAttributeFlags,
		},
		{
			name: "nested bit, type 1, length 0",
			attrs: []Attribute{
				{
					Length: 4,
					Type:   1,
					Nested: true,
					Data:   make([]byte, 0),
				},
			},
			b: []byte{
				0x04, 0x00,
				0x01, 0x80, // Nested bit
			},
		},
		{
			name: "endianness bit, type 1, length 0",
			attrs: []Attribute{
				{
					Length:       4,
					Type:         1,
					NetByteOrder: true,
					Data:         make([]byte, 0),
				},
			},
			b: []byte{
				0x04, 0x00,
				0x01, 0x40, // NetByteOrder bit
			},
		},
		{
			name: "max type space, length 0",
			attrs: []Attribute{
				{
					Length:       4,
					Type:         16383,
					Nested:       false,
					NetByteOrder: false,
					Data:         make([]byte, 0),
				},
			},
			b: []byte{
				0x04, 0x00,
				0xFF, 0x3F, // 14 lowest type bits up
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
			name: "nested and endianness bits",
			b: []byte{
				0x04, 0x00,
				0x00, 0xC0, // both bits set
			},
			err: errInvalidAttributeFlags,
		},
		{
			name: "nested bit, type 1, length 0",
			b: []byte{
				0x04, 0x00,
				0x01, 0x80, // Nested bit
			},
			attrs: []Attribute{
				{
					Length: 4,
					Type:   1,
					Nested: true,
					Data:   make([]byte, 0),
				},
			},
		},
		{
			name: "endianness bit, type 1, length 0",
			b: []byte{
				0x04, 0x00,
				0x01, 0x40, // NetByteOrder bit
			},
			attrs: []Attribute{
				{
					Length:       4,
					Type:         1,
					NetByteOrder: true,
					Data:         make([]byte, 0),
				},
			},
		},
		{
			name: "max type space, length 0",
			b: []byte{
				0x04, 0x00,
				0xFF, 0x3F, // 14 lowest type bits up
			},
			attrs: []Attribute{
				{
					Length:       4,
					Type:         16383,
					Nested:       false,
					NetByteOrder: false,
					Data:         make([]byte, 0),
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
