package nlenc

import (
	"bytes"
	"fmt"
	"testing"
)

func TestUintPanic(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		fn   func(b []byte)
	}{
		{
			name: "short put 16",
			b:    make([]byte, 1),
			fn: func(b []byte) {
				PutUint16(b, 0)
			},
		},
		{
			name: "long put 16",
			b:    make([]byte, 3),
			fn: func(b []byte) {
				PutUint16(b, 0)
			},
		},
		{
			name: "short get 16",
			b:    make([]byte, 1),
			fn: func(b []byte) {
				Uint16(b)
			},
		},
		{
			name: "long get 16",
			b:    make([]byte, 3),
			fn: func(b []byte) {
				Uint16(b)
			},
		},
		{
			name: "short put 32",
			b:    make([]byte, 3),
			fn: func(b []byte) {
				PutUint32(b, 0)
			},
		},
		{
			name: "long put 32",
			b:    make([]byte, 5),
			fn: func(b []byte) {
				PutUint32(b, 0)
			},
		},
		{
			name: "short get 32",
			b:    make([]byte, 3),
			fn: func(b []byte) {
				Uint32(b)
			},
		},
		{
			name: "long get 32",
			b:    make([]byte, 5),
			fn: func(b []byte) {
				Uint32(b)
			},
		},
		{
			name: "short get signed 32",
			b:    make([]byte, 3),
			fn: func(b []byte) {
				Int32(b)
			},
		},
		{
			name: "long get signed 32",
			b:    make([]byte, 5),
			fn: func(b []byte) {
				Int32(b)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("expected panic, but none occurred")
				}
			}()

			tt.fn(tt.b)
			t.Fatal("reached end of test case without panic")
		})
	}
}

func TestUint16(t *testing.T) {
	tests := []struct {
		v uint16
		b []byte
	}{
		{
			v: 0x1,
			b: []byte{0x01, 0x00},
		},
		{
			v: 0x0102,
			b: []byte{0x02, 0x01},
		},
		{
			v: 0x1234,
			b: []byte{0x34, 0x12},
		},
		{
			v: 0xffff,
			b: []byte{0xff, 0xff},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("0x%04x", tt.v), func(t *testing.T) {
			b := make([]byte, 2)
			PutUint16(b, tt.v)

			if want, got := tt.b, b; !bytes.Equal(want, got) {
				t.Fatalf("unexpected bytes:\n- want: [%# x]\n-  got: [%# x]",
					want, got)
			}

			v := Uint16(b)

			if want, got := tt.v, v; want != got {
				t.Fatalf("unexpected integer:\n- want: 0x%04x\n-  got: 0x%04x",
					want, got)
			}
		})
	}
}

func TestUint32(t *testing.T) {
	tests := []struct {
		v uint32
		b []byte
	}{
		{
			v: 0x1,
			b: []byte{0x01, 0x00, 0x00, 0x00},
		},
		{
			v: 0x0102,
			b: []byte{0x02, 0x01, 0x00, 0x00},
		},
		{
			v: 0x1234,
			b: []byte{0x34, 0x12, 0x00, 0x00},
		},
		{
			v: 0xffff,
			b: []byte{0xff, 0xff, 0x00, 0x00},
		},
		{
			v: 0x01020304,
			b: []byte{0x04, 0x03, 0x02, 0x01},
		},
		{
			v: 0x1a2a3a4a,
			b: []byte{0x4a, 0x3a, 0x2a, 0x1a},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("0x%04x", tt.v), func(t *testing.T) {
			b := make([]byte, 4)
			PutUint32(b, tt.v)

			if want, got := tt.b, b; !bytes.Equal(want, got) {
				t.Fatalf("unexpected bytes:\n- want: [%# x]\n-  got: [%# x]",
					want, got)
			}

			v := Uint32(b)

			if want, got := tt.v, v; want != got {
				t.Fatalf("unexpected integer:\n- want: 0x%04x\n-  got: 0x%04x",
					want, got)
			}
		})
	}
}

func TestInt32(t *testing.T) {
	tests := []struct {
		b []byte
		v int32
	}{
		{
			b: []byte{0x01, 0x00, 0x00, 0x00},
			v: 0x1,
		},
		{
			b: []byte{0x02, 0x01, 0x00, 0x00},
			v: 0x0102,
		},
		{
			b: []byte{0x34, 0x12, 0x00, 0x00},
			v: 0x1234,
		},
		{
			b: []byte{0xff, 0xff, 0x00, 0x00},
			v: 0xffff,
		},
		{
			b: []byte{0x04, 0x03, 0x02, 0x01},
			v: 0x01020304,
		},
		{
			b: []byte{0x4a, 0x3a, 0x2a, 0x1a},
			v: 0x1a2a3a4a,
		},
		{
			b: []byte{0xff, 0xff, 0xff, 0xff},
			v: -1,
		},
		{
			b: []byte{0xfe, 0xff, 0xff, 0xff},
			v: -2,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("0x%04x", tt.v), func(t *testing.T) {
			v := Int32(tt.b)

			if want, got := tt.v, v; want != got {
				t.Fatalf("unexpected integer:\n- want: 0x%04x\n-  got: 0x%04x",
					want, got)
			}
		})
	}
}
