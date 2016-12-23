package netlink

import (
	"errors"
)

// Various errors which may occur when attempting to marshal or unmarshal
// a Message to and from its binary form.
var (
	errIncorrectMessageLength = errors.New("netlink message header length incorrect")
	errShortMessage           = errors.New("not enough data to create a netlink message")
	errUnalignedMessage       = errors.New("input data is not properly aligned for netlink message")
)

// HeaderFlags specify flags which may be present in a Header.
type HeaderFlags uint16

const (
	// General netlink communication flags.

	// HeaderFlagsRequest indicates a request to netlink.
	HeaderFlagsRequest HeaderFlags = 1

	// HeaderFlagsMulti indicates a multi-part message, terminated
	// by HeaderTypeDone on the last message.
	HeaderFlagsMulti HeaderFlags = 2

	// HeaderFlagsAcknowledge requests that netlink reply with
	// an acknowledgement using HeaderTypeError and, if needed,
	// an error code.
	HeaderFlagsAcknowledge HeaderFlags = 4

	// HeaderFlagsEcho requests that netlink echo this request
	// back to the sender.
	HeaderFlagsEcho HeaderFlags = 8

	// HeaderFlagsDumpInterrupted indicates that a dump was
	// inconsistent due to a sequence change.
	HeaderFlagsDumpInterrupted HeaderFlags = 16

	// HeaderFlagsDumpFiltered indicates that a dump was filtered
	// as requested.
	HeaderFlagsDumpFiltered HeaderFlags = 32

	// Flags used to retrieve data from netlink.

	// HeaderFlagsRoot requests that netlink return a complete table instead
	// of a single entry.
	HeaderFlagsRoot HeaderFlags = 0x100

	// HeaderFlagsMatch requests that netlink return a list of all matching
	// entries.
	HeaderFlagsMatch HeaderFlags = 0x200

	// HeaderFlagsAtomic requests that netlink send an atomic snapshot of
	// its entries.  Requires CAP_NET_ADMIN or an effective UID of 0.
	// May be obsolete.
	HeaderFlagsAtomic HeaderFlags = 0x300

	// HeaderFlagsDump requests that netlink return a complete list of
	// all entries.
	HeaderFlagsDump HeaderFlags = HeaderFlagsRoot | HeaderFlagsMatch
)

// HeaderType specifies the type of a Header.
type HeaderType uint16

const (
	// HeaderTypeNoop indicates that no action was taken.
	HeaderTypeNoop HeaderType = 0x1

	// HeaderTypeError indicates an error code is present, which is also
	// used to indicate success when the code is 0.
	HeaderTypeError HeaderType = 0x2

	// HeaderTypeDone indicates the end of a multi-part message.
	HeaderTypeDone HeaderType = 0x3

	// HeaderTypeOverrun indicates that data was lost from this message.
	HeaderTypeOverrun HeaderType = 0x4
)

// NB: the memory layout of Header and Linux's syscall.NlMsgHdr must be
// exactly the same.  Cannot reorder, change data type, add, or remove fields.
// Named types of the same size (e.g. HeaderFlags is a uint16) are okay.

// A Header is a netlink header.  A Header is sent and received with each
// Message to indicate metadata regarding a Message.
type Header struct {
	// Length of a Message, including this Header.
	Length uint32

	// Contents of a Message.
	Type HeaderType

	// Flags which may be used to modify a request or response.
	Flags HeaderFlags

	// The sequence number of a Message.
	Sequence uint32

	// The process ID of the sending process.
	PID uint32
}

// A Message is a netlink message.  It contains a Header and an arbitrary
// byte payload, which may be decoded using information from the Header.
//
// Data is encoded in the native endianness of the host system.  Use this
// package's Uint* and PutUint* functions to encode and decode integers.
type Message struct {
	Header Header
	Data   []byte
}

// MarshalBinary marshals a Message into a byte slice.
func (m Message) MarshalBinary() ([]byte, error) {
	ml := nlmsgAlign(int(m.Header.Length))
	if ml < nlmsgHeaderLen || ml != int(m.Header.Length) {
		return nil, errIncorrectMessageLength
	}

	b := make([]byte, ml)

	PutUint32(b[0:4], m.Header.Length)
	PutUint16(b[4:6], uint16(m.Header.Type))
	PutUint16(b[6:8], uint16(m.Header.Flags))
	PutUint32(b[8:12], m.Header.Sequence)
	PutUint32(b[12:16], m.Header.PID)
	copy(b[16:], m.Data)

	return b, nil
}

// UnmarshalBinary unmarshals the contents of a byte slice into a Message.
func (m *Message) UnmarshalBinary(b []byte) error {
	if len(b) < nlmsgHeaderLen {
		return errShortMessage
	}
	if len(b) != nlmsgAlign(len(b)) {
		return errUnalignedMessage
	}

	// Don't allow misleading length
	m.Header.Length = Uint32(b[0:4])
	if int(m.Header.Length) != len(b) {
		return errShortMessage
	}

	m.Header.Type = HeaderType(Uint16(b[4:6]))
	m.Header.Flags = HeaderFlags(Uint16(b[6:8]))
	m.Header.Sequence = Uint32(b[8:12])
	m.Header.PID = Uint32(b[12:16])
	m.Data = b[16:]

	return nil
}
