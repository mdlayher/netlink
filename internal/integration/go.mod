module github.com/mdlayher/netlink/internal/integration

go 1.25.0

require (
	github.com/google/go-cmp v0.7.0
	github.com/google/nftables v0.3.1-0.20251119083706-1db35da82052
	github.com/jsimonetti/rtnetlink v1.4.2
	github.com/mdlayher/ethtool v0.6.0
	github.com/vishvananda/netns v0.0.5
	golang.org/x/net v0.52.0
	golang.org/x/sys v0.42.0
)

require (
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/socket v0.6.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
)

// We require a recent release, but in reality the integration tests should
// always use the netlink module at the root of the repository.
require github.com/mdlayher/netlink v1.11.0

replace github.com/mdlayher/netlink => ../../
