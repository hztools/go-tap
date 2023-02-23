// {{{ Copyright (c) Paul R. Tagliamonte <paul@k3xec.com>, 2022
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE. }}}

//go:build linux

package tap

import (
	"net"

	"github.com/vishvananda/netlink"
)

// AddNeighbor will add an entry into the ARP / NDP table
func (i Interface) AddNeighbor(mac net.HardwareAddr, ip net.IP) error {
	if ip4 := ip.To4(); len(ip4) == net.IPv4len {
		return i.addNeighborIPv4(mac, ip4)
	}
	return i.addNeighborIPv6(mac, ip)
}

func (i Interface) addNeighborIPv4(mac net.HardwareAddr, ip net.IP) error {
	iface := i.ps.netif

	return netlink.NeighAdd(&netlink.Neigh{
		LinkIndex:    iface.Attrs().Index,
		Type:         netlink.FAMILY_V4,
		State:        netlink.NUD_PERMANENT,
		IP:           ip,
		HardwareAddr: mac,
	})
}

func (i Interface) addNeighborIPv6(mac net.HardwareAddr, ip net.IP) error {
	iface := i.ps.netif

	return netlink.NeighAdd(&netlink.Neigh{
		LinkIndex:    iface.Attrs().Index,
		Type:         netlink.FAMILY_V6,
		State:        netlink.NUD_PERMANENT,
		IP:           ip,
		HardwareAddr: mac,
	})
}

// vim: foldmethod=marker
