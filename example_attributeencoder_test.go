package netlink_test

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/mdlayher/netlink"
)

// encodeNested is a nested structure within out.
type encodeNested struct {
	A, B uint32
}

// encodeOut is an example structure we will use to pack netlink attributes.
type encodeOut struct {
	Number uint16
	String string
	Nested encodeNested
}

// encode is an example function used to adapt the ae.Do method
// to encode an arbitrary structure.
func (n encodeNested) encode() func() ([]byte, error) {
	return func() ([]byte, error) {
		// Create an internal, nested netlink.NewAttributeEncoder that
		// operates on the nested set of attributes.
		ae := netlink.NewAttributeEncoder()

		// Encode the fields of the nested structure.
		ae.Uint32(1, n.A)
		ae.Uint32(2, n.B)

		// Return the encoded attributes, and any error encountered.
		return ae.Encode()
	}
}

func ExampleAttributeEncoder_encode() {
	// Create a netlink.AttributeEncoder that encodes to the same message
	// as that decoded by the netlink.AttributeDecoder example.
	ae := netlink.NewAttributeEncoder()

	o := encodeOut{
		Number: 1,
		String: "hello world",
		Nested: encodeNested{
			A: 2,
			B: 3,
		},
	}

	// Encode the Number attribute as a uint16.
	ae.Uint16(1, o.Number)
	// Encode the String attribute as a string.
	ae.String(2, o.String)
	// Nested is a nested structure, so we will use the encodeNested type's
	// encode method with ae.Do to encode it in a concise way.
	ae.Do(3, o.Nested.encode())

	// Any errors encountered during encoding (including any errors from
	// encoding nested attributes) will be returned here.
	b, err := ae.Encode()
	if err != nil {
		log.Fatalf("failed to encode attributes: %v", err)
	}

	fmt.Printf("Encoded netlink message follows:\n%s", hex.Dump(b))

	// Output: Encoded netlink message follows:
	// 00000000  06 00 01 00 01 00 00 00  10 00 02 00 68 65 6c 6c  |............hell|
	// 00000010  6f 20 77 6f 72 6c 64 00  14 00 03 00 08 00 01 00  |o world.........|
	// 00000020  02 00 00 00 08 00 02 00  03 00 00 00              |............|
}
