module github.com/mdlayher/netlink

go 1.25.0

retract (
	// contains a bug where `netlink.Conn.Receive` blocks concurrent calls to
	// `netlink.Conn.Send` when blocking in `recvmsg`.
	v1.11.1
	// contains a bug where `netlink.Conn.Receive` and `netlink.Conn.ReceiveIter` reject
	// valid netlink messages whose header length is not aligned.
	v1.11.0
	// contains a critical bug where `netlink.Conn.Receive` and `netlink.Conn.ReceiveIter`
	// would panic if the received message was unaligned.
	v1.10.0
)

require (
	github.com/google/go-cmp v0.7.0
	github.com/mdlayher/socket v0.6.1
	golang.org/x/net v0.56.0
	golang.org/x/sys v0.46.0
)

require golang.org/x/sync v0.20.0 // indirect
