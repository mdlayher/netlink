package netlink_test

import (
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

// encode is an example function used to adapt the ae.Nested method
// to encode an arbitrary structure.
func (n encodeNested) encode(ae *netlink.AttributeEncoder) error {
	// Encode the fields of the nested structure.
	ae.Uint32(1, n.A)
	ae.Uint32(2, n.B)
	return nil
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
	// encode method with ae.Nested to encode it in a concise way.
	ae.Nested(3, o.Nested.encode)

	// Any errors encountered during encoding (including any errors from
	// encoding nested attributes) will be returned here.
	b, err := ae.Encode()
	if err != nil {
		log.Fatalf("failed to encode attributes: %v", err)
	}

	// Now decode the attributes again to verify the contents.
	ad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		log.Fatalf("failed to decode attributes: %v", err)
	}

	// Walk the attributes and print each out.
	for ad.Next() {
		switch ad.Type() {
		case 1:
			fmt.Println("uint16:", ad.Uint16())
		case 2:
			fmt.Println("string:", ad.String())
		case 3:
			fmt.Println("nested:")

			// Nested attributes use their own nested decoder.
			ad.Nested(func(nad *netlink.AttributeDecoder) error {
				for nad.Next() {
					switch nad.Type() {
					case 1:
						fmt.Println("  - A:", nad.Uint32())
					case 2:
						fmt.Println("  - B:", nad.Uint32())
					}
				}
				return nil
			})
		}
	}

	// Output: uint16: 1
	// string: hello world
	// nested:
	//   - A: 2
	//   - B: 3
}
