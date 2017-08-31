package genetlink_test

import (
	"log"
	"net"
	"os"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/genetlink"
	"github.com/mdlayher/netlink/nlenc"
)

// This example demonstrates using a genetlink.Conn's high level interface
// to query for a specific generic netlink family.
func ExampleConn_getFamily() {
	c, err := genetlink.Dial(nil)
	if err != nil {
		log.Fatalf("failed to dial generic netlink: %v", err)
	}
	defer c.Close()

	// Ask generic netlink about the generic netlink controller (nlctrl)'s
	// family information
	const name = "nlctrl"
	family, err := c.GetFamily(name)
	if err != nil {
		// If family doesn't exist, error can be checked using os.IsNotExist
		if os.IsNotExist(err) {
			log.Printf("%q family not available", name)
			return
		}

		log.Fatalf("failed to query for family: %v", err)
	}

	log.Printf("%s: %+v", name, family)
}

// This example demonstrates using a genetlink.Conn's high level interface
// to query for all known generic netlink families.
func ExampleConn_listFamilies() {
	c, err := genetlink.Dial(nil)
	if err != nil {
		log.Fatalf("failed to dial generic netlink: %v", err)
	}
	defer c.Close()

	// Ask generic netlink about all families registered with it
	families, err := c.ListFamilies()
	if err != nil {
		log.Fatalf("failed to query for families: %v", err)
	}

	for i, f := range families {
		log.Printf("#%02d: %+v", i, f)
	}
}

// This example demonstrates using a genetlink.Conn's high level and low
// level interfaces to detect if nl80211 (netlink 802.11 WiFi device
// generic netlink family) is available, and if it is, sending a request
// to it to retrieve all WiFi interfaces.
func ExampleConn_nl80211WiFi() {
	c, err := genetlink.Dial(nil)
	if err != nil {
		log.Fatalf("failed to dial generic netlink: %v", err)
	}
	defer c.Close()

	// Constants which are sourced from nl80211.h.
	const (
		name = "nl80211"

		nl80211CommandGetInterface = 5

		nl80211AttributeInterfaceIndex = 3
		nl80211AttributeInterfaceName  = 4
		nl80211AttributeAttributeMAC   = 6
	)

	// Ask generic netlink if nl80211 is available
	family, err := c.GetFamily(name)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("%q family not available", name)
			return
		}

		log.Fatalf("failed to query for family: %v", err)
	}

	// Ask nl80211 to dump a list of all WiFi interfaces
	req := genetlink.Message{
		Header: genetlink.Header{
			Command: nl80211CommandGetInterface,
			Version: family.Version,
		},
	}

	// Send request specifically to nl80211 instead of generic netlink
	// controller (nlctrl)
	flags := netlink.HeaderFlagsRequest | netlink.HeaderFlagsDump
	msgs, err := c.Execute(req, family.ID, flags)
	if err != nil {
		log.Fatalf("failed to execute: %v", err)
	}

	// Some basic information about a WiFi interface
	type ifInfo struct {
		Index int
		Name  string
		MAC   net.HardwareAddr
	}

	var infos []ifInfo
	for _, m := range msgs {
		// nl80211's response contains packed netlink attributes
		attrs, err := netlink.UnmarshalAttributes(m.Data)
		if err != nil {
			log.Fatalf("failed to unmarshal attributes: %v", err)
		}

		// Gather data about interface from attributes
		var info ifInfo
		for _, a := range attrs {
			switch a.Type {
			case nl80211AttributeInterfaceIndex:
				info.Index = int(nlenc.Uint32(a.Data))
			case nl80211AttributeInterfaceName:
				info.Name = nlenc.String(a.Data)
			case nl80211AttributeAttributeMAC:
				info.MAC = net.HardwareAddr(a.Data)
			}
		}

		infos = append(infos, info)
	}

	for i, info := range infos {
		log.Printf("#%02d: %+v", i, info)
	}
}
