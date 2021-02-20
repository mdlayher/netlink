package nlenc

import (
	"fmt"
	"unsafe"
)

// PutUint8 encodes a uint8 into b.
// If b is not exactly 1 byte in length, PutUint8 will panic.
func PutUint8(b []byte, v uint8) {
	if l := len(b); l != 1 {
		panic(fmt.Sprintf("PutUint8: unexpected byte slice length: %d", l))
	}

	b[0] = v
}

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

// PutUint64 encodes a uint64 into b using the host machine's native endianness.
// If b is not exactly 8 bytes in length, PutUint64 will panic.
func PutUint64(b []byte, v uint64) {
	if l := len(b); l != 8 {
		panic(fmt.Sprintf("PutUint64: unexpected byte slice length: %d", l))
	}

	*(*uint64)(unsafe.Pointer(&b[0])) = v
}

// PutInt8 encodes a int8 into b using the host machine's native endianness.
// If b is not exactly 1 bytes in length, PutInt8 will panic.
func PutInt8(b []byte, v int8) {
	if l := len(b); l != 1 {
		panic(fmt.Sprintf("PutInt8: unexpected byte slice length: %d", l))
	}

	*(*int8)(unsafe.Pointer(&b[0])) = v
}

// PutInt16 encodes a int16 into b using the host machine's native endianness.
// If b is not exactly 2 bytes in length, PutInt16 will panic.
func PutInt16(b []byte, v int16) {
	if l := len(b); l != 2 {
		panic(fmt.Sprintf("PutInt16: unexpected byte slice length: %d", l))
	}

	*(*int16)(unsafe.Pointer(&b[0])) = v
}

// PutInt32 encodes a int32 into b using the host machine's native endianness.
// If b is not exactly 4 bytes in length, PutInt32 will panic.
func PutInt32(b []byte, v int32) {
	if l := len(b); l != 4 {
		panic(fmt.Sprintf("PutInt32: unexpected byte slice length: %d", l))
	}

	*(*int32)(unsafe.Pointer(&b[0])) = v
}

// PutInt64 encodes a int64 into b using the host machine's native endianness.
// If b is not exactly 8 bytes in length, PutInt64 will panic.
func PutInt64(b []byte, v int64) {
	if l := len(b); l != 8 {
		panic(fmt.Sprintf("PutInt64: unexpected byte slice length: %d", l))
	}

	*(*int64)(unsafe.Pointer(&b[0])) = v
}

// Uint8 decodes a uint8 from b.
// If b is not exactly 1 byte in length, Uint8 will panic.
func Uint8(b []byte) uint8 {
	if l := len(b); l != 1 {
		panic(fmt.Sprintf("Uint8: unexpected byte slice length: %d", l))
	}

	return b[0]
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

// Uint64 decodes a uint64 from b using the host machine's native endianness.
// If b is not exactly 8 bytes in length, Uint64 will panic.
func Uint64(b []byte) uint64 {
	if l := len(b); l != 8 {
		panic(fmt.Sprintf("Uint64: unexpected byte slice length: %d", l))
	}

	return *(*uint64)(unsafe.Pointer(&b[0]))
}

// Int8 decodes an int8 from b using the host machine's native endianness.
// If b is not exactly 1 bytes in length, Int8 will panic.
func Int8(b []byte) int8 {
	if l := len(b); l != 1 {
		panic(fmt.Sprintf("Int8: unexpected byte slice length: %d", l))
	}

	return *(*int8)(unsafe.Pointer(&b[0]))
}

// Int16 decodes an int16 from b using the host machine's native endianness.
// If b is not exactly 2 bytes in length, Int16 will panic.
func Int16(b []byte) int16 {
	if l := len(b); l != 2 {
		panic(fmt.Sprintf("Int16: unexpected byte slice length: %d", l))
	}

	return *(*int16)(unsafe.Pointer(&b[0]))
}

// Int32 decodes an int32 from b using the host machine's native endianness.
// If b is not exactly 4 bytes in length, Int32 will panic.
func Int32(b []byte) int32 {
	if l := len(b); l != 4 {
		panic(fmt.Sprintf("Int32: unexpected byte slice length: %d", l))
	}

	return *(*int32)(unsafe.Pointer(&b[0]))
}

// Int64 decodes an int64 from b using the host machine's native endianness.
// If b is not exactly 8 bytes in length, Int64 will panic.
func Int64(b []byte) int64 {
	if l := len(b); l != 8 {
		panic(fmt.Sprintf("Int64: unexpected byte slice length: %d", l))
	}

	return *(*int64)(unsafe.Pointer(&b[0]))
}

// Uint8Bytes encodes a uint8 into a newly-allocated byte slice. It is a
// shortcut for allocating a new byte slice and filling it using PutUint8.
func Uint8Bytes(v uint8) []byte {
	b := make([]byte, 1)
	PutUint8(b, v)
	return b
}

// Uint16Bytes encodes a uint16 into a newly-allocated byte slice using the
// host machine's native endianness.  It is a shortcut for allocating a new
// byte slice and filling it using PutUint16.
func Uint16Bytes(v uint16) []byte {
	b := make([]byte, 2)
	PutUint16(b, v)
	return b
}

// Uint32Bytes encodes a uint32 into a newly-allocated byte slice using the
// host machine's native endianness.  It is a shortcut for allocating a new
// byte slice and filling it using PutUint32.
func Uint32Bytes(v uint32) []byte {
	b := make([]byte, 4)
	PutUint32(b, v)
	return b
}

// Uint64Bytes encodes a uint64 into a newly-allocated byte slice using the
// host machine's native endianness.  It is a shortcut for allocating a new
// byte slice and filling it using PutUint64.
func Uint64Bytes(v uint64) []byte {
	b := make([]byte, 8)
	PutUint64(b, v)
	return b
}

// Int8Bytes encodes a int8 into a newly-allocated byte slice using the
// host machine's native endianness.  It is a shortcut for allocating a new
// byte slice and filling it using PutInt8.
func Int8Bytes(v int8) []byte {
	b := make([]byte, 1)
	PutInt8(b, v)
	return b
}

// Int16Bytes encodes a int16 into a newly-allocated byte slice using the
// host machine's native endianness.  It is a shortcut for allocating a new
// byte slice and filling it using PutInt16.
func Int16Bytes(v int16) []byte {
	b := make([]byte, 2)
	PutInt16(b, v)
	return b
}

// Int32Bytes encodes a int32 into a newly-allocated byte slice using the
// host machine's native endianness.  It is a shortcut for allocating a new
// byte slice and filling it using PutInt32.
func Int32Bytes(v int32) []byte {
	b := make([]byte, 4)
	PutInt32(b, v)
	return b
}

// Int64Bytes encodes a int64 into a newly-allocated byte slice using the
// host machine's native endianness.  It is a shortcut for allocating a new
// byte slice and filling it using PutInt64.
func Int64Bytes(v int64) []byte {
	b := make([]byte, 8)
	PutInt64(b, v)
	return b
}
