package netlink_test

import (
	"fmt"
	"log"

	"github.com/mdlayher/netlink"
)

// decodeNested is a nested structure within decodeOut.
type decodeNested struct {
	A, B uint32
}

// decodeOut is an example structure we will use to unpack netlink attributes.
type decodeOut struct {
	Number uint16
	String string
	Nested decodeNested
}

// decode is an example function used to adapt the ad.Do method to decode an
// arbitrary structure.
func (n *decodeNested) decode() func(b []byte) error {
	return func(b []byte) error {
		// Create an internal netlink.AttributeDecoder that operates on a
		// nested set of attributes passed by the external decoder.
		ad, err := netlink.NewAttributeDecoder(b)
		if err != nil {
			return fmt.Errorf("failed to create nested attribute decoder: %v", err)
		}

		// Iterate attributes until completion, checking the type of each
		// and decoding them as appropriate.
		for ad.Next() {
			switch ad.Type() {
			// A and B are both uint32 values, so decode them as such.
			case 1:
				n.A = ad.Uint32()
			case 2:
				n.B = ad.Uint32()
			}
		}

		// Return any error encountered while decoding.
		return ad.Err()
	}
}

// This example demonstrates using a netlink.AttributeDecoder to decode packed
// netlink attributes in a message payload.
func ExampleAttributeDecoder_decode() {
	// Create a netlink.AttributeDecoder using some example attribute bytes
	// that are prepared for this example.
	ad, err := netlink.NewAttributeDecoder(exampleAttributes())
	if err != nil {
		log.Fatalf("failed to create attribute decoder: %v", err)
	}

	// Iterate attributes until completion, checking the type of each and
	// decoding them as appropriate.
	var out decodeOut
	for ad.Next() {
		// Check the type of the current attribute with ad.Type.  Typically you
		// will find netlink attribute types and data values in C headers as
		// constants.
		switch ad.Type() {
		case 1:
			// Number is a uint16.
			out.Number = ad.Uint16()
		case 2:
			// String is a string.
			out.String = ad.String()
		case 3:
			// Nested is a nested structure, so we will use a method on the
			// nested type along with ad.Do to decode it in a concise way.
			ad.Do(out.Nested.decode())
		}
	}

	// Any errors encountered during decoding (including any errors from
	// decoding the nested attributes) will be returned here.
	if err := ad.Err(); err != nil {
		log.Fatalf("failed to decode attributes: %v", err)
	}

	fmt.Printf(`Number: %d
String: %q
Nested:
   - A: %d
   - B: %d`,
		out.Number, out.String, out.Nested.A, out.Nested.B,
	)
	// Output:
	// Number: 1
	// String: "hello world"
	// Nested:
	//    - A: 2
	//    - B: 3
}
