# netlink [![Test Status](https://github.com/mdlayher/netlink/workflows/Linux%20Test/badge.svg)](https://github.com/mdlayher/netlink/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/mdlayher/netlink.svg)](https://pkg.go.dev/github.com/mdlayher/netlink)  [![Go Report Card](https://goreportcard.com/badge/github.com/mdlayher/netlink)](https://goreportcard.com/report/github.com/mdlayher/netlink)

Package `netlink` provides low-level access to Linux netlink sockets.
MIT Licensed.

For more information about how netlink works, check out my blog series
on [Linux, Netlink, and Go](https://mdlayher.com/blog/linux-netlink-and-go-part-1-netlink/).

If you have any questions or you'd like some guidance, please join us on
[Gophers Slack](https://invite.slack.golangbridge.org) in the `#networking`
channel!

## Stability

See the [CHANGELOG](./CHANGELOG.md) file for a description of changes between
releases.

This package has a stable v1 API and any future breaking changes will prompt
the release of a new major version. Features and bug fixes will continue to
occur in the v1.x.x series.

In order to reduce the maintenance burden, this package is only supported on
Go 1.12+. Older versions of Go lack critical features and APIs which are
necessary for this package to function correctly.

**If you depend on this package in your applications, please use Go modules.**

## Design

A [number of netlink packages](https://godoc.org/?q=netlink) are already
available for Go, but I wasn't able to find one that aligned with what
I wanted in a netlink package:

- Straightforward, idiomatic API
- Well tested
- Well documented
- Doesn't use package/global variables or state
- Doesn't necessarily need root to work

My goal for this package is to use it as a building block for the creation
of other netlink family packages.

## Ecosystem

Over time, an ecosystem of Go packages has developed around package `netlink`.
Many of these packages provide building blocks for further interactions with
various netlink families, such as `NETLINK_GENERIC` or `NETLINK_ROUTE`.

To have your package included in this diagram, please send a pull request!

``` mermaid
flowchart LR
    netlink["https://github.com/mdlayher/netlink"]

    subgraph "NETLINK_CONNECTOR"
        direction LR

        garlic["github.com/fearful-symmetry/garlic"]
    end

    subgraph "NETLINK_CRYPTO"
        direction LR

        cryptonl["github.com/fearful-symmetry/garlic"]
    end

    subgraph "NETLINK_GENERIC"
        direction LR

        genetlink["github.com/mdlayher/genetlink"]

        devlink["github.com/mdlayher/devlink"]
        ethtool["github.com/mdlayher/ethtool"]
        go-openvswitch["github.com/digitalocean/go-openvswitch"]
        ipvs["github.com/cloudflare/ipvs"]
        l2tp["github.com/axatrax/l2tp"]
        ndb["github.com/Merovius/nbd"]
        quota["github.com/mdlayher/quota"]
        router7["github.com/rtr7/router7"]
        taskstats["github.com/mdlayher/taskstats"]
        u-bmc["github.com/u-root/u-bmc"]
        wgctl["golang.zx2c4.com/wireguard/wgctrl"]
        wifi["github.com/mdlayher/wifi"]

        devlink & ethtool & go-openvswitch & ipvs --> genetlink
        l2tp & ndb & quota & router7 & taskstats --> genetlink
        u-bmc & wgctl & wifi --> genetlink
    end

    subgraph "NETLINK_KOBJECT_UEVENT"
        direction LR

        kobject["github.com/mdlayher/kobject"]
    end

    subgraph "NETLINK_NETFILTER"
        direction LR

        go-conntrack["github.com/florianl/go-conntrack"]
        go-nflog["github.com/florianl/go-nflog"]
        go-nfqueue["github.com/florianl/go-nfqueue"]
        netfilter["github.com/ti-mo/netfilter"]
        nftables["github.com/google/nftables"]

        conntrack["github.com/ti-mo/conntrack"]

        conntrack --> netfilter
    end

    subgraph "NETLINK_ROUTE"
        direction LR

        go-tc["github.com/florianl/go-tc"]
        qdisc["github.com/ema/qdisc"]
        rtnetlink["github.com/jsimonetti/rtnetlink"]
        rtnl["gitlab.com/mergetb/tech/rtnl"]
    end

    subgraph "NETLINK_W1"
        direction LR

        go-onewire["github.com/SpComb/go-onewire"]
    end

    NETLINK_CONNECTOR --> netlink
    NETLINK_CRYPTO --> netlink
    NETLINK_GENERIC --> netlink
    NETLINK_KOBJECT_UEVENT --> netlink
    NETLINK_NETFILTER --> netlink
    NETLINK_ROUTE --> netlink
    NETLINK_W1 --> netlink
```
