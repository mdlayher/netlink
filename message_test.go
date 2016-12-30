package netlink

import (
	"bytes"
	"reflect"
	"testing"
)

// TODO(mdlayher): tests all assume little endian host machine

func TestHeaderFlagsString(t *testing.T) {
	tests := []struct {
		f HeaderFlags
		s string
	}{
		{
			f: 0,
			s: "0",
		},
		{
			f: HeaderFlagsRequest,
			s: "request",
		},
		{
			f: HeaderFlagsMulti,
			s: "multi",
		},
		{
			f: HeaderFlagsEcho,
			s: "echo",
		},
		{
			f: HeaderFlagsDumpInterrupted,
			s: "dumpinterrupted",
		},
		{
			f: HeaderFlagsDumpFiltered,
			s: "dumpfiltered",
		},
		{
			f: 1 << 6,
			s: "1<<6",
		},
		{
			f: 1 << 7,
			s: "1<<7",
		},
		{
			f: HeaderFlagsRoot,
			s: "root",
		},
		{
			f: HeaderFlagsMatch,
			s: "match",
		},
		{
			f: HeaderFlagsAtomic,
			s: "atomic",
		},
		{
			f: HeaderFlagsDump,
			s: "root|match",
		},
		{
			f: HeaderFlagsRequest | HeaderFlagsDump,
			s: "request|root|match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if want, got := tt.s, tt.f.String(); want != got {
				t.Fatalf("unexpected flag string for: %016b\n- want: %q\n-  got: %q",
					tt.f, want, got)
			}
		})
	}
}

func TestHeaderTypeString(t *testing.T) {
	tests := []struct {
		t HeaderType
		s string
	}{
		{
			t: 0,
			s: "unknown(0)",
		},
		{
			t: HeaderTypeNoop,
			s: "noop",
		},
		{
			t: HeaderTypeError,
			s: "error",
		},
		{
			t: HeaderTypeDone,
			s: "done",
		},
		{
			t: HeaderTypeOverrun,
			s: "overrun",
		},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if want, got := tt.s, tt.t.String(); want != got {
				t.Fatalf("unexpected header type string:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestMessageMarshal(t *testing.T) {
	tests := []struct {
		name string
		m    Message
		b    []byte
		err  error
	}{
		{
			name: "empty",
			m:    Message{},
			err:  errIncorrectMessageLength,
		},
		{
			name: "short",
			m: Message{
				Header: Header{
					Length: 15,
				},
			},
			err: errIncorrectMessageLength,
		},
		{
			name: "unaligned",
			m: Message{
				Header: Header{
					Length: 17,
				},
			},
			err: errIncorrectMessageLength,
		},
		{
			name: "OK no data",
			m: Message{
				Header: Header{
					Length: 16,
				},
			},
			b: []byte{
				0x10, 0x00, 0x00, 0x00,
				0x00, 0x00,
				0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "OK unaligned data",
			m: Message{
				Header: Header{
					Length:   20,
					Flags:    HeaderFlagsRequest,
					Sequence: 1,
					PID:      10,
				},
				Data: []byte("abc"),
			},
			b: []byte{
				0x14, 0x00, 0x00, 0x00,
				0x00, 0x00,
				0x01, 0x00,
				0x01, 0x00, 0x00, 0x00,
				0x0a, 0x00, 0x00, 0x00,
				0x61, 0x62, 0x63, 0x00, /* last byte padded */
			},
		},
		{
			name: "OK aligned data",
			m: Message{
				Header: Header{
					Length:   20,
					Type:     HeaderTypeError,
					Sequence: 2,
					PID:      20,
				},
				Data: []byte("abcd"),
			},
			b: []byte{
				0x14, 0x00, 0x00, 0x00,
				0x02, 0x00,
				0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x14, 0x00, 0x00, 0x00,
				0x61, 0x62, 0x63, 0x64,
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

func TestMessageUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		m    Message
		err  error
	}{
		{
			name: "empty",
			err:  errShortMessage,
		},
		{
			name: "short",
			b:    make([]byte, 15),
			err:  errShortMessage,
		},
		{
			name: "unaligned",
			b:    make([]byte, 17),
			err:  errUnalignedMessage,
		},
		{
			name: "fuzz crasher: length shorter than slice",
			b:    []byte("\x1d000000000000000"),
			err:  errShortMessage,
		},
		{
			name: "fuzz crasher: length longer than slice",
			b:    []byte("\x13\x00\x00\x000000000000000000"),
			err:  errShortMessage,
		},
		{
			name: "OK no data",
			b: []byte{
				0x10, 0x00, 0x00, 0x00,
				0x00, 0x00,
				0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
			m: Message{
				Header: Header{
					Length: 16,
				},
				Data: make([]byte, 0),
			},
		},
		{
			name: "OK data",
			m: Message{
				Header: Header{
					Length:   20,
					Type:     HeaderTypeError,
					Sequence: 2,
					PID:      20,
				},
				Data: []byte("abcd"),
			},
			b: []byte{
				0x14, 0x00, 0x00, 0x00,
				0x02, 0x00,
				0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
				0x14, 0x00, 0x00, 0x00,
				0x61, 0x62, 0x63, 0x64,
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
