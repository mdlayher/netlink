module github.com/mdlayher/netlink/internal/integration

go 1.24.0

require (
	github.com/google/go-cmp v0.7.0
	github.com/jsimonetti/rtnetlink v1.4.2
	github.com/mdlayher/ethtool v0.4.1
	golang.org/x/net v0.46.0
	golang.org/x/sys v0.37.0
)

require (
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	golang.org/x/sync v0.3.0 // indirect
)

// We require a recent release, but in reality the integration tests should
// always use the netlink module at the root of the repository.
require github.com/mdlayher/netlink v1.8.0

replace github.com/mdlayher/netlink => ../../
