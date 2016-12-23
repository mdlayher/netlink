package genetlink

import (
	"bytes"
	"reflect"
	"testing"
)

func TestMessageMarshalBinary(t *testing.T) {
	tests := []struct {
		name string
		m    Message
		b    []byte
		err  error
	}{
		{
			name: "empty",
			m:    Message{},
			b:    []byte{0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "no data",
			m: Message{
				Header: Header{
					Command: 1,
					Version: 2,
				},
			},
			b: []byte{0x01, 0x02, 0x00, 0x00},
		},
		{
			name: "data",
			m: Message{
				Header: Header{
					Command: 1,
					Version: 2,
				},
				Data: []byte{0x03, 0x04},
			},
			b: []byte{
				0x01, 0x02, 0x00, 0x00,
				0x03, 0x04,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.m.MarshalBinary()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v", want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.b, b; !bytes.Equal(want, got) {
				t.Fatalf("unexpected Message bytes:\n- want: [%# x]\n-  got: [%# x]", want, got)
			}
		})
	}
}

func TestMessageUnmarshalBinary(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		m    Message
		err  error
	}{
		{
			name: "empty",
			err:  errInvalidMessage,
		},
		{
			name: "short",
			b:    make([]byte, 3),
			err:  errInvalidMessage,
		},
		{
			name: "1st reserved set",
			b:    []byte{0x00, 0x00, 0x01, 0x00},
			err:  errInvalidMessage,
		},
		{
			name: "2nd reserved set",
			b:    []byte{0x00, 0x00, 0x00, 0x01},
			err:  errInvalidMessage,
		},
		{
			name: "both reserved set",
			b:    []byte{0x00, 0x00, 0x01, 0x01},
			err:  errInvalidMessage,
		},
		{
			name: "zero value",
			b:    []byte{0x00, 0x00, 0x00, 0x00},
			m: Message{
				Data: make([]byte, 0),
			},
		},
		{
			name: "no data",
			b:    []byte{0x01, 0x02, 0x00, 0x00},
			m: Message{
				Header: Header{
					Command: 1,
					Version: 2,
				},
				Data: make([]byte, 0),
			},
		},
		{
			name: "data",
			b: []byte{
				0x01, 0x02, 0x00, 0x00,
				0x03, 0x04,
			},
			m: Message{
				Header: Header{
					Command: 1,
					Version: 2,
				},
				Data: []byte{0x03, 0x04},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Message
			err := (&m).UnmarshalBinary(tt.b)

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v", want, got)
			}
			if err != nil {
				return
			}

			if want, got := tt.m, m; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected Message:\n- want: %#v\n-  got: %#v", want, got)
			}
		})
	}
}
