package netlink

import (
	"fmt"
	"unsafe"
)

// PutUint16 encodes a uint16 into b using the host machine's native endianness.
// If b is not exactly 2 bytes in length, PutUint16 will panic.
func PutUint16(b []byte, v uint16) {
	if l := len(b); l != 2 {
		panic(fmt.Sprintf("PutUint16: unexpected byte slice length: %d", l))
	}

	*(*uint16)(unsafe.Pointer(&b[0])) = v
}

// PutUint32 encodes a uint32 into b using the host machine's native endianness.
// If b is not exactly 4 bytes in length, PutUint32 will panic.
func PutUint32(b []byte, v uint32) {
	if l := len(b); l != 4 {
		panic(fmt.Sprintf("PutUint32: unexpected byte slice length: %d", l))
	}

	*(*uint32)(unsafe.Pointer(&b[0])) = v
}

// Uint16 decodes a uint16 from b using the host machine's native endianness.
// If b is not exactly 2 bytes in length, Uint16 will panic.
func Uint16(b []byte) uint16 {
	if l := len(b); l != 2 {
		panic(fmt.Sprintf("Uint16: unexpected byte slice length: %d", l))
	}

	return *(*uint16)(unsafe.Pointer(&b[0]))
}

// Uint32 decodes a uint32 from b using the host machine's native endianness.
// If b is not exactly 4 bytes in length, Uint32 will panic.
func Uint32(b []byte) uint32 {
	if l := len(b); l != 4 {
		panic(fmt.Sprintf("Uint32: unexpected byte slice length: %d", l))
	}

	return *(*uint32)(unsafe.Pointer(&b[0]))
}

// getInt32 decodes an int32 from b using the host machine's native endianness.
// If b is not exactly 4 bytes in length, getInt32 will panic.
//
// getInt32 is named as such to prevent collision with the built-in int32 type.
func getInt32(b []byte) int32 {
	if l := len(b); l != 4 {
		panic(fmt.Sprintf("getInt32: unexpected byte slice length: %d", l))
	}

	return *(*int32)(unsafe.Pointer(&b[0]))
}
