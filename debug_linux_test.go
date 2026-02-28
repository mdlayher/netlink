//go:build linux
// +build linux

package netlink

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/sys/unix"
)

func TestNlmsgFprintf(t *testing.T) {
	// netlink messages obtained from https://github.com/google/nftables/blob/e99829fb4f26d75fdd0cfce8ba4632744e72c2bc/nftables_test.go#L245C1-L246C94
	tests := []struct {
		name     string
		m        Message
		colorize bool
		want     string
	}{
		{
			name: "nft add table ip nat",
			m: Message{
				Header: Header{
					Length:   40,
					Type:     HeaderType(uint16(unix.NFNL_SUBSYS_NFTABLES)<<8 | uint16(unix.NFT_MSG_NEWTABLE)),
					Flags:    Request,
					Sequence: 1,
					PID:      1234,
				},
				Data: []byte("\x02\x00\x00\x00\x08\x00\x01\x00\x6e\x61\x74\x00\x08\x00\x02\x00\x00\x00\x00\x00"),
			},
			colorize: false,
			want: `----------------	------------------
|  0000000040  |	| message length |
| 02560 | R--- |	|  type | flags  |
|  0000000001  |	| sequence number|
|  0000001234  |	|     port ID    |
----------------	------------------
| 02 00 00 00  |	|  extra header  |
|00008|--|00001|	|len |flags| type|
| 6e 61 74 00  |	|      data      |	 n a t  
|00008|--|00002|	|len |flags| type|
| 00 00 00 00  |	|      data      |	        
----------------	------------------
`,
		},
		{
			name: "nft add rule nat prerouting iifname uplink0 udp dport 4070-4090 dnat 192.168.23.2:4070-4090",
			m: Message{
				Header: Header{
					Length:   40,
					Type:     HeaderType(uint16(unix.NFNL_SUBSYS_NFTABLES)<<8 | uint16(unix.NFT_MSG_NEWRULE)),
					Flags:    Request,
					Sequence: 1,
					PID:      1234,
				},
				Data: []byte("\x02\x00\x00\x00\x08\x00\x01\x00\x6e\x61\x74\x00\x0f\x00\x02\x00\x70\x72\x65\x72\x6f\x75\x74\x69\x6e\x67\x00\x00\xf8\x01\x04\x80\x24\x00\x01\x80\x09\x00\x01\x00\x6d\x65\x74\x61\x00\x00\x00\x00\x14\x00\x02\x80\x08\x00\x02\x00\x00\x00\x00\x06\x08\x00\x01\x00\x00\x00\x00\x01\x38\x00\x01\x80\x08\x00\x01\x00\x63\x6d\x70\x00\x2c\x00\x02\x80\x08\x00\x01\x00\x00\x00\x00\x01\x08\x00\x02\x00\x00\x00\x00\x00\x18\x00\x03\x80\x14\x00\x01\x00\x75\x70\x6c\x69\x6e\x6b\x30\x00\x00\x00\x00\x00\x00\x00\x00\x00\x24\x00\x01\x80\x09\x00\x01\x00\x6d\x65\x74\x61\x00\x00\x00\x00\x14\x00\x02\x80\x08\x00\x02\x00\x00\x00\x00\x10\x08\x00\x01\x00\x00\x00\x00\x01\x2c\x00\x01\x80\x08\x00\x01\x00\x63\x6d\x70\x00\x20\x00\x02\x80\x08\x00\x01\x00\x00\x00\x00\x01\x08\x00\x02\x00\x00\x00\x00\x00\x0c\x00\x03\x80\x05\x00\x01\x00\x11\x00\x00\x00\x34\x00\x01\x80\x0c\x00\x01\x00\x70\x61\x79\x6c\x6f\x61\x64\x00\x24\x00\x02\x80\x08\x00\x01\x00\x00\x00\x00\x01\x08\x00\x02\x00\x00\x00\x00\x02\x08\x00\x03\x00\x00\x00\x00\x02\x08\x00\x04\x00\x00\x00\x00\x02\x2c\x00\x01\x80\x08\x00\x01\x00\x63\x6d\x70\x00\x20\x00\x02\x80\x08\x00\x01\x00\x00\x00\x00\x01\x08\x00\x02\x00\x00\x00\x00\x05\x0c\x00\x03\x80\x06\x00\x01\x00\x0f\xe6\x00\x00\x2c\x00\x01\x80\x08\x00\x01\x00\x63\x6d\x70\x00\x20\x00\x02\x80\x08\x00\x01\x00\x00\x00\x00\x01\x08\x00\x02\x00\x00\x00\x00\x03\x0c\x00\x03\x80\x06\x00\x01\x00\x0f\xfa\x00\x00\x2c\x00\x01\x80\x0e\x00\x01\x00\x69\x6d\x6d\x65\x64\x69\x61\x74\x65\x00\x00\x00\x18\x00\x02\x80\x08\x00\x01\x00\x00\x00\x00\x01\x0c\x00\x02\x80\x08\x00\x01\x00\xc0\xa8\x17\x02\x2c\x00\x01\x80\x0e\x00\x01\x00\x69\x6d\x6d\x65\x64\x69\x61\x74\x65\x00\x00\x00\x18\x00\x02\x80\x08\x00\x01\x00\x00\x00\x00\x02\x0c\x00\x02\x80\x06\x00\x01\x00\x0f\xe6\x00\x00\x2c\x00\x01\x80\x0e\x00\x01\x00\x69\x6d\x6d\x65\x64\x69\x61\x74\x65\x00\x00\x00\x18\x00\x02\x80\x08\x00\x01\x00\x00\x00\x00\x03\x0c\x00\x02\x80\x06\x00\x01\x00\x0f\xfa\x00\x00\x38\x00\x01\x80\x08\x00\x01\x00\x6e\x61\x74\x00\x2c\x00\x02\x80\x08\x00\x01\x00\x00\x00\x00\x01\x08\x00\x02\x00\x00\x00\x00\x02\x08\x00\x03\x00\x00\x00\x00\x01\x08\x00\x05\x00\x00\x00\x00\x02\x08\x00\x06\x00\x00\x00\x00\x03"),
			},
			colorize: false,
			want: `----------------	------------------
|  0000000040  |	| message length |
| 02566 | R--- |	|  type | flags  |
|  0000000001  |	| sequence number|
|  0000001234  |	|     port ID    |
----------------	------------------
| 02 00 00 00  |	|  extra header  |
|00008|--|00001|	|len |flags| type|
| 6e 61 74 00  |	|      data      |	 n a t  
|00015|--|00002|	|len |flags| type|
| 70 72 65 72  |	|      data      |	 p r e r
| 6f 75 74 69  |	|      data      |	 o u t i
| 6e 67 00 00  |	|      data      |	 n g    
|00504|N-|00004|	|len |flags| type|
|00036|N-|00001|	|len |flags| type|
|00009|--|00001|	|len |flags| type|
| 6d 65 74 61  |	|      data      |	 m e t a
| 00 00 00 00  |	|      data      |	        
|00020|N-|00002|	|len |flags| type|
|00008|--|00002|	|len |flags| type|
| 00 00 00 06  |	|      data      |	        
|00008|--|00001|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00056|N-|00001|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 63 6d 70 00  |	|      data      |	 c m p  
|00044|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00008|--|00002|	|len |flags| type|
| 00 00 00 00  |	|      data      |	        
|00024|N-|00003|	|len |flags| type|
|00020|--|00001|	|len |flags| type|
| 75 70 6c 69  |	|      data      |	 u p l i
| 6e 6b 30 00  |	|      data      |	 n k 0  
| 00 00 00 00  |	|      data      |	        
| 00 00 00 00  |	|      data      |	        
|00036|N-|00001|	|len |flags| type|
|00009|--|00001|	|len |flags| type|
| 6d 65 74 61  |	|      data      |	 m e t a
| 00 00 00 00  |	|      data      |	        
|00020|N-|00002|	|len |flags| type|
|00008|--|00002|	|len |flags| type|
| 00 00 00 10  |	|      data      |	        
|00008|--|00001|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00044|N-|00001|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 63 6d 70 00  |	|      data      |	 c m p  
|00032|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00008|--|00002|	|len |flags| type|
| 00 00 00 00  |	|      data      |	        
|00012|N-|00003|	|len |flags| type|
|00005|--|00001|	|len |flags| type|
| 11 00 00 00  |	|      data      |	        
|00052|N-|00001|	|len |flags| type|
|00012|--|00001|	|len |flags| type|
| 70 61 79 6c  |	|      data      |	 p a y l
| 6f 61 64 00  |	|      data      |	 o a d  
|00036|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00008|--|00002|	|len |flags| type|
| 00 00 00 02  |	|      data      |	        
|00008|--|00003|	|len |flags| type|
| 00 00 00 02  |	|      data      |	        
|00008|--|00004|	|len |flags| type|
| 00 00 00 02  |	|      data      |	        
|00044|N-|00001|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 63 6d 70 00  |	|      data      |	 c m p  
|00032|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00008|--|00002|	|len |flags| type|
| 00 00 00 05  |	|      data      |	        
|00012|N-|00003|	|len |flags| type|
|00006|--|00001|	|len |flags| type|
| 0f e6 00 00  |	|      data      |	   æ    
|00044|N-|00001|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 63 6d 70 00  |	|      data      |	 c m p  
|00032|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00008|--|00002|	|len |flags| type|
| 00 00 00 03  |	|      data      |	        
|00012|N-|00003|	|len |flags| type|
|00006|--|00001|	|len |flags| type|
| 0f fa 00 00  |	|      data      |	   ú    
|00044|N-|00001|	|len |flags| type|
|00014|--|00001|	|len |flags| type|
| 69 6d 6d 65  |	|      data      |	 i m m e
| 64 69 61 74  |	|      data      |	 d i a t
| 65 00 00 00  |	|      data      |	 e      
|00024|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00012|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| c0 a8 17 02  |	|      data      |	 À ¨    
|00044|N-|00001|	|len |flags| type|
|00014|--|00001|	|len |flags| type|
| 69 6d 6d 65  |	|      data      |	 i m m e
| 64 69 61 74  |	|      data      |	 d i a t
| 65 00 00 00  |	|      data      |	 e      
|00024|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 00 00 00 02  |	|      data      |	        
|00012|N-|00002|	|len |flags| type|
|00006|--|00001|	|len |flags| type|
| 0f e6 00 00  |	|      data      |	   æ    
|00044|N-|00001|	|len |flags| type|
|00014|--|00001|	|len |flags| type|
| 69 6d 6d 65  |	|      data      |	 i m m e
| 64 69 61 74  |	|      data      |	 d i a t
| 65 00 00 00  |	|      data      |	 e      
|00024|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 00 00 00 03  |	|      data      |	        
|00012|N-|00002|	|len |flags| type|
|00006|--|00001|	|len |flags| type|
| 0f fa 00 00  |	|      data      |	   ú    
|00056|N-|00001|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 6e 61 74 00  |	|      data      |	 n a t  
|00044|N-|00002|	|len |flags| type|
|00008|--|00001|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00008|--|00002|	|len |flags| type|
| 00 00 00 02  |	|      data      |	        
|00008|--|00003|	|len |flags| type|
| 00 00 00 01  |	|      data      |	        
|00008|--|00005|	|len |flags| type|
| 00 00 00 02  |	|      data      |	        
|00008|--|00006|	|len |flags| type|
| 00 00 00 03  |	|      data      |	        
----------------	------------------
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			nlmsgFprintf(w, tt.m, tt.colorize)
			got := w.String()
			if got != tt.want {
				t.Errorf("nlmsgFprintf() =\n%s,\nwant\n%s\ndiff:\n%s", got, tt.want, cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestNlmsgFprintfHeader(t *testing.T) {
	tests := []struct {
		name string
		h    Header
		want string
	}{
		{
			name: "Basic test",
			h: Header{
				Length:   16 + 4,
				Type:     0,
				Flags:    Request,
				Sequence: 1,
				PID:      123,
			},
			want: `----------------	------------------
|  0000000020  |	| message length |
| 00000 | R--- |	|  type | flags  |
|  0000000001  |	| sequence number|
|  0000000123  |	|     port ID    |
----------------	------------------
`,
		},
		{
			name: "All flags",
			h: Header{
				Length:   16,
				Type:     unix.NFNL_SUBSYS_IPSET << 8,
				Flags:    Request | Multi | Acknowledge | Echo | Dump | DumpFiltered | Create | Excl | Append,
				Sequence: 123,
				PID:      456,
			},
			want: `----------------	------------------
|  0000000016  |	| message length |
| 01536 | RMAE |	|  type | flags  |
|  0000000123  |	| sequence number|
|  0000000456  |	|     port ID    |
----------------	------------------
`,
		},
		{
			name: "Unknown type",
			h: Header{
				Length:   16,
				Type:     0xffff,
				Flags:    Request | Acknowledge,
				Sequence: 123,
				PID:      456,
			},
			want: `----------------	------------------
|  0000000016  |	| message length |
| 65535 | R-A- |	|  type | flags  |
|  0000000123  |	| sequence number|
|  0000000456  |	|     port ID    |
----------------	------------------
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			nlmsgFprintfHeader(w, tt.h)
			if got := w.String(); got != tt.want {
				t.Errorf("nlmsgFprintfHeader() =\n%s,\nwant\n%s\ndiff:\n%s", got, tt.want, cmp.Diff(got, tt.want))
			}
		})
	}
}
