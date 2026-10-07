//go:build linux

package netlink

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/sys/unix"
)

type debugTestAttribute func(*AttributeEncoder)

func debugStringAttribute(typ uint16, value string) debugTestAttribute {
	return func(ae *AttributeEncoder) { ae.String(typ, value) }
}

func debugUint32Attribute(typ uint16, value uint32) debugTestAttribute {
	return func(ae *AttributeEncoder) { ae.Uint32(typ, value) }
}

func debugBytesAttribute(typ uint16, value []byte) debugTestAttribute {
	return func(ae *AttributeEncoder) { ae.Bytes(typ, value) }
}

func debugNestedAttribute(typ uint16, attrs ...debugTestAttribute) debugTestAttribute {
	return func(ae *AttributeEncoder) {
		ae.Nested(typ, func(nae *AttributeEncoder) error {
			for _, attr := range attrs {
				attr(nae)
			}
			return nil
		})
	}
}

func debugExpression(name string, attrs ...debugTestAttribute) debugTestAttribute {
	return debugNestedAttribute(
		unix.NFTA_LIST_ELEM,
		debugStringAttribute(unix.NFTA_EXPR_NAME, name),
		debugNestedAttribute(unix.NFTA_EXPR_DATA, attrs...),
	)
}

func debugMetaExpression(key, dreg uint32) debugTestAttribute {
	return debugExpression(
		"meta",
		debugUint32Attribute(unix.NFTA_META_KEY, key),
		debugUint32Attribute(unix.NFTA_META_DREG, dreg),
	)
}

func debugCmpExpression(sreg, op uint32, value []byte) debugTestAttribute {
	return debugExpression(
		"cmp",
		debugUint32Attribute(unix.NFTA_CMP_SREG, sreg),
		debugUint32Attribute(unix.NFTA_CMP_OP, op),
		debugNestedAttribute(
			unix.NFTA_CMP_DATA,
			debugBytesAttribute(unix.NFTA_DATA_VALUE, value),
		),
	)
}

func debugPayloadExpression(dreg, base, offset, length uint32) debugTestAttribute {
	return debugExpression(
		"payload",
		debugUint32Attribute(unix.NFTA_PAYLOAD_DREG, dreg),
		debugUint32Attribute(unix.NFTA_PAYLOAD_BASE, base),
		debugUint32Attribute(unix.NFTA_PAYLOAD_OFFSET, offset),
		debugUint32Attribute(unix.NFTA_PAYLOAD_LEN, length),
	)
}

func debugImmediateExpression(dreg uint32, value []byte) debugTestAttribute {
	return debugExpression(
		"immediate",
		debugUint32Attribute(unix.NFTA_IMMEDIATE_DREG, dreg),
		debugNestedAttribute(
			unix.NFTA_IMMEDIATE_DATA,
			debugBytesAttribute(unix.NFTA_DATA_VALUE, value),
		),
	)
}

func debugNatExpression(
	typ, family, addressRegister, portMinRegister, portMaxRegister uint32,
) debugTestAttribute {
	return debugExpression(
		"nat",
		debugUint32Attribute(unix.NFTA_NAT_TYPE, typ),
		debugUint32Attribute(unix.NFTA_NAT_FAMILY, family),
		debugUint32Attribute(unix.NFTA_NAT_REG_ADDR_MIN, addressRegister),
		debugUint32Attribute(unix.NFTA_NAT_REG_PROTO_MIN, portMinRegister),
		debugUint32Attribute(unix.NFTA_NAT_REG_PROTO_MAX, portMaxRegister),
	)
}

func debugBigEndianUint16(value uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, value)
	return b
}

func debugNftablesMessageData(t *testing.T, attrs ...debugTestAttribute) []byte {
	t.Helper()

	ae := NewAttributeEncoder()

	// nftables integer attribute payloads use network byte order. Attribute
	// headers are written in native byte order by MarshalAttributes.
	ae.ByteOrder = binary.BigEndian

	for _, attr := range attrs {
		attr(ae)
	}

	b, err := ae.Encode()
	if err != nil {
		t.Fatalf("failed to encode nftables attributes: %v", err)
	}

	return append([]byte{unix.NFPROTO_IPV4, 0, 0, 0}, b...)
}

func TestNlmsgFprintf(t *testing.T) {
	// nftables messages modeled after https://github.com/google/nftables/blob/e99829fb4f26d75fdd0cfce8ba4632744e72c2bc/nftables_test.go#L245C1-L246C94
	tests := []struct {
		name     string
		m        func(t *testing.T) Message
		colorize bool
		want     string
	}{
		{
			name: "nft add table ip nat",
			m: func(t *testing.T) Message {
				return Message{
					Header: Header{
						Length:   40,
						Type:     HeaderType(uint16(unix.NFNL_SUBSYS_NFTABLES)<<8 | uint16(unix.NFT_MSG_NEWTABLE)),
						Flags:    Request,
						Sequence: 1,
						PID:      1234,
					},
					Data: debugNftablesMessageData(
						t,
						debugStringAttribute(unix.NFTA_TABLE_NAME, "nat"),
						debugUint32Attribute(unix.NFTA_TABLE_FLAGS, 0),
					),
				}
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
			m: func(t *testing.T) Message {
				return Message{
					Header: Header{
						Length:   40,
						Type:     HeaderType(uint16(unix.NFNL_SUBSYS_NFTABLES)<<8 | uint16(unix.NFT_MSG_NEWRULE)),
						Flags:    Request,
						Sequence: 1,
						PID:      1234,
					},
					Data: debugNftablesMessageData(
						t,
						debugStringAttribute(unix.NFTA_RULE_TABLE, "nat"),
						debugStringAttribute(unix.NFTA_RULE_CHAIN, "prerouting"),
						debugNestedAttribute(
							unix.NFTA_RULE_EXPRESSIONS,

							debugMetaExpression(
								unix.NFT_META_IIFNAME,
								unix.NFT_REG_1,
							),

							debugCmpExpression(
								unix.NFT_REG_1,
								unix.NFT_CMP_EQ,
								append([]byte("uplink0"), make([]byte, 9)...),
							),

							debugMetaExpression(
								unix.NFT_META_L4PROTO,
								unix.NFT_REG_1,
							),

							debugCmpExpression(
								unix.NFT_REG_1,
								unix.NFT_CMP_EQ,
								[]byte{unix.IPPROTO_UDP},
							),

							debugPayloadExpression(
								unix.NFT_REG_1,
								unix.NFT_PAYLOAD_TRANSPORT_HEADER,
								2,
								2,
							),

							debugCmpExpression(
								unix.NFT_REG_1,
								unix.NFT_CMP_GTE,
								debugBigEndianUint16(4070),
							),

							debugCmpExpression(
								unix.NFT_REG_1,
								unix.NFT_CMP_LTE,
								debugBigEndianUint16(4090),
							),

							debugImmediateExpression(
								unix.NFT_REG_1,
								[]byte{192, 168, 23, 2},
							),

							debugImmediateExpression(
								unix.NFT_REG_2,
								debugBigEndianUint16(4070),
							),

							debugImmediateExpression(
								unix.NFT_REG_3,
								debugBigEndianUint16(4090),
							),

							debugNatExpression(
								unix.NFT_NAT_DNAT,
								unix.NFPROTO_IPV4,
								unix.NFT_REG_1,
								unix.NFT_REG_2,
								unix.NFT_REG_3,
							),
						),
					),
				}
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
			m := tt.m(t)
			nlmsgFprintf(w, m, tt.colorize)
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
