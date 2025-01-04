package netlink

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/mdlayher/netlink/nlenc"
)

// Arguments used to create a debugger.
var debugArgs []string

func init() {
	// Is netlink debugging enabled?
	s := os.Getenv("NLDEBUG")
	if s == "" {
		return
	}

	debugArgs = strings.Split(s, ",")
}

// A debugger is used to provide debugging information about a netlink connection.
type debugger struct {
	Log    *log.Logger
	Level  int
	Format string
}

// newDebugger creates a debugger by parsing key=value arguments.
func newDebugger(args []string) *debugger {
	d := &debugger{
		Log:   log.New(os.Stderr, "nl: ", 0),
		Level: 1,
	}

	for _, a := range args {
		kv := strings.Split(a, "=")
		if len(kv) != 2 {
			// Ignore malformed pairs and assume callers wants defaults.
			continue
		}

		switch kv[0] {
		// Select the log level for the debugger.
		case "level":
			level, err := strconv.Atoi(kv[1])
			if err != nil {
				panicf("netlink: invalid NLDEBUG level: %q", a)
			}

			d.Level = level
		case "format":
			d.Format = kv[1]
		}
	}

	return d
}

// debugf prints debugging information at the specified level, if d.Level is
// high enough to print the message.
func (d *debugger) debugf(level int, format string, v ...interface{}) {
	if d.Level >= level {
		if d.Format == "mnl" {
			for _, iface := range v {
				if msg, ok := iface.(Message); ok {
					nlmsgFprintf(d.Log.Writer(), msg)
				}
			}
		} else {
			d.Log.Printf(format, v...)
		}
	}
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

/*
	nlmsgFprintf - print netlink message to file
	- Based on https://git.netfilter.org/libmnl/tree/src/nlmsg.c
	- This function prints the netlink header to a file handle.
	- It may be useful for debugging purposes. One example of the output
	- is the following:

----------------        ------------------
|  0000000040  |        | message length |
| 00016 | R-A- |        |  type | flags  |
|  1289148991  |        | sequence number|
|  0000000000  |        |     port ID    |
----------------        ------------------
| 00 00 00 00  |        |  extra header  |
| 00 00 00 00  |        |  extra header  |
| 01 00 00 00  |        |  extra header  |
| 01 00 00 00  |        |  extra header  |
|00008|--|00003|        |len |flags| type|
| 65 74 68 30  |        |      data      |       e t h 0
----------------        ------------------

	*
	* This example above shows the netlink message that is send to kernel-space
	* to set up the link interface eth0. The netlink and attribute header data
	* are displayed in base 10 whereas the extra header and the attribute payload
	* are expressed in base 16. The possible flags in the netlink header are:
	*
	* - R, that indicates that NLM_F_REQUEST is set.
	* - M, that indicates that NLM_F_MULTI is set.
	* - A, that indicates that NLM_F_ACK is set.
	* - E, that indicates that NLM_F_ECHO is set.
	*
	* The lack of one flag is displayed with '-'. On the other hand, the possible
	* attribute flags available are:
	*
	* - N, that indicates that NLA_F_NESTED is set.
	* - B, that indicates that NLA_F_NET_BYTEORDER is set.
*/
func nlmsgFprintfHeader(fd io.Writer, nlh Header) {
	fmt.Fprintf(fd, "----------------\t------------------\n")
	fmt.Fprintf(fd, "|  %010d  |\t| message length |\n", nlh.Length)
	fmt.Fprintf(fd, "| %05d | %s%s%s%s |\t|  type | flags  |\n",
		nlh.Type,
		ternary(nlh.Flags&Request != 0, "R", "-"),
		ternary(nlh.Flags&Multi != 0, "M", "-"),
		ternary(nlh.Flags&Acknowledge != 0, "A", "-"),
		ternary(nlh.Flags&Echo != 0, "E", "-"),
	)
	fmt.Fprintf(fd, "|  %010d   |\t| sequence number|\n", nlh.Sequence)
	fmt.Fprintf(fd, "|  %010d  |\t|     port ID    |\n", nlh.PID)
	fmt.Fprintf(fd, "----------------\t------------------\n")
}

// nlmsgFprintf checks a single Message for netlink errors.
func nlmsgFprintf(fd io.Writer, m Message) {
	colorize := true
	var hasHeader bool
	nlmsgFprintfHeader(fd, m.Header)
	switch {
	case m.Header.Type == Error:
		hasHeader = true
	case m.Header.Type == Done && m.Header.Flags&Multi != 0:
		if len(m.Data) == 0 {
			return
		}
	default:
		// Neither, nothing to do.
	}

	// Errno occupies 4 bytes.
	const endErrno = 4
	if len(m.Data) < endErrno {
		return
	}

	c := nlenc.Int32(m.Data[:endErrno])
	if c != 0 {
		b := m.Data[0:4]
		fmt.Fprintf(fd, "| %.2x %.2x %.2x %.2x  |\t",
			0xff&b[0], 0xff&b[1],
			0xff&b[2], 0xff&b[3])
		fmt.Fprintf(fd, "|  extra header  |\n")
	}

	// Flags indicate an extended acknowledgement. The type/flags combination
	// checked above determines the offset where the TLVs occur.
	var off int
	if hasHeader {
		// There is an nlmsghdr preceding the TLVs.
		if len(m.Data) < endErrno+nlmsgHeaderLen {
			return
		}

		// The TLVs should be at the offset indicated by the nlmsghdr.length,
		// plus the offset where the header began. But make sure the calculated
		// offset is still in-bounds.
		h := *(*Header)(unsafe.Pointer(&m.Data[endErrno : endErrno+nlmsgHeaderLen][0]))
		off = endErrno + int(h.Length)

		if len(m.Data) < off {
			return
		}
	} else {
		// There is no nlmsghdr preceding the TLVs, parse them directly.
		off = endErrno
	}

	data := m.Data[off:]
	for i := 0; i < len(data); {
		// Make sure there's at least a header's worth
		// of data to read on each iteration.
		if len(data[i:]) < nlaHeaderLen {
			break
		}

		// Extract the length of the attribute.
		l := int(nlenc.Uint16(data[i : i+2]))
		// extract the type
		t := nlenc.Uint16(data[i+2 : i+4])
		// print attribute header
		if colorize {
			fmt.Fprintf(fd, "|\033[1;31m%05d|\033[1;32m%s%s|\033[1;34m%05d\033[0m|\t",
				l,
				ternary(t&syscall.NLA_F_NESTED != 0, "N", "-"),
				ternary(t&syscall.NLA_F_NET_BYTEORDER != 0, "B", "-"),
				t&attrTypeMask)
			fmt.Fprintf(fd, "|len |flags| type|\n")
		} else {
			fmt.Fprintf(fd, "|%05d|%s%s|%05d|\t",
				l,
				ternary(t&syscall.NLA_F_NESTED != 0, "N", "-"),
				ternary(t&syscall.NLA_F_NET_BYTEORDER != 0, "B", "-"),
				t&attrTypeMask)
			fmt.Fprintf(fd, "|len |flags| type|\n")
		}

		nextAttr := i + nlaAlign(l)

		// advance the pointer to the bytes after the header
		i += nlaHeaderLen

		// Ignore zero-length attributes.
		if l == 0 {
			continue
		}
		// If nested check the next attribute
		if t&syscall.NLA_F_NESTED != 0 {
			continue
		}

		// Print the remaining attributes bytes
		for ; i < nextAttr; i += 4 {
			fmt.Fprintf(fd, "| %.2x %.2x %.2x %.2x  |\t",
				0xff&data[i], 0xff&data[i+1],
				0xff&data[i+2], 0xff&data[i+3])

			fmt.Fprintf(fd, "|      data      |")

			fmt.Fprintf(fd, "\t %s %s %s %s\n",
				ternary(strconv.IsPrint(rune(data[i])), string(data[i]), " "),
				ternary(strconv.IsPrint(rune(data[i+1])), string(data[i+1]), " "),
				ternary(strconv.IsPrint(rune(data[i+2])), string(data[i+2]), " "),
				ternary(strconv.IsPrint(rune(data[i+3])), string(data[i+3]), " "),
			)
		}
	}
	fmt.Fprintf(fd, "----------------\t------------------\n")
}

func ternary(cond bool, iftrue string, iffalse string) string {
	if cond {
		return iftrue
	} else {
		return iffalse
	}
}
