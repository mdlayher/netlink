package nlenc

import "testing"

func TestBytesString(t *testing.T) {
	tests := []struct {
		s string
		b []byte
	}{
		{
			s: "foo",
			b: []byte{'f', 'o', 'o', 0x00},
		},
		{
			s: "nl80211",
			b: []byte{'n', 'l', '8', '0', '2', '1', '1', 0x00},
		},
		{
			s: "TASKSTATS",
			b: []byte{'T', 'A', 'S', 'K', 'S', 'T', 'A', 'T', 'S', 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			s := String(Bytes(tt.s))

			if want, got := tt.s, s; want != got {
				t.Fatalf("unexpected string:\n- want: %q\n-  got: %q", want, got)
			}
		})
	}
}
