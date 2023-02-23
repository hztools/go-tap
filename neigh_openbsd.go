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

//go:build openbsd

package tap

import (
	"net"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Address order for RTA_* additions.
//
// RTA_DST                           = 0x1
// RTA_GATEWAY                       = 0x2
// RTA_NETMASK                       = 0x4
// RTA_GENMASK                       = 0x8
// RTA_IFP                           = 0x10
// RTA_IFA                           = 0x20
// RTA_AUTHOR                        = 0x40
// RTA_BRD                           = 0x80
// RTA_SRC                           = 0x100
// RTA_SRCMASK                       = 0x200
// RTA_LABEL                         = 0x400

const (
	sizeofRtMsgData = unix.SizeofSockaddrAny * 4
	sizeofRtMsg     = unix.SizeofRtMsghdr + sizeofRtMsgData
)

type rtMsg struct {
	hdr  unix.RtMsghdr
	data [sizeofRtMsgData]byte
}

// newNeighRtMsg will create a RTM_ADD rtMsg, which expects two addresses,
// the RTA_DST, and the RTA_GATEWAY. This assumes a HOST (non-local) and
// STATIC route (ARP/NDP).
func newNeighRtMsg() ([]byte, *rtMsg) {
	rtMsgBytes := make([]byte, sizeofRtMsg)
	msg := (*rtMsg)(unsafe.Pointer(&rtMsgBytes[0]))
	hdr := &msg.hdr
	hdr.Version = unix.RTM_VERSION
	hdr.Type = unix.RTM_ADD
	hdr.Flags = unix.RTF_HOST | unix.RTF_STATIC
	hdr.Addrs = 0
	hdr.Pid = int32(os.Getpid())
	hdr.Inits = unix.RTV_EXPIRE
	hdr.Seq = 1
	return rtMsgBytes, msg
}

func (i Interface) addNeighborIPv4(mac net.HardwareAddr, ip net.IP) error {
	netif := i.ps.netif
	rtMsgBytes, msg := newNeighRtMsg()
	msg.hdr.Index = uint16(netif.Index)

	// TODO(paultag): Expose some knobs to allow for expiring routes.
	// hdr.Rmx = unix.RtMetrics{
	// 	Expire: time.Now().Add(time.Second * 10).Unix(),
	// }

	idx := 0
	ip4 := rawSockaddrInet4(ip.To4())
	msg.hdr.Addrs |= unix.RTA_DST
	*(*unix.RawSockaddrInet4)(unsafe.Pointer(&msg.data[idx])) = ip4
	idx += int(ip4.Len + (ip4.Len % 8))

	dl := rawSockaddrDatalink(mac)
	dl.Index = uint16(netif.Index)
	msg.hdr.Addrs |= unix.RTA_GATEWAY
	*(*unix.RawSockaddrDatalink)(unsafe.Pointer(&msg.data[idx])) = dl
	idx += int(dl.Len + (dl.Len % 8))

	msg.hdr.Msglen = uint16(unix.SizeofRtMsghdr + idx)
	msg.hdr.Hdrlen = uint16(unix.SizeofRtMsghdr)

	file, err := afRoute()
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(rtMsgBytes[:msg.hdr.Msglen])
	return err
}

func (i Interface) addNeighborIPv6(mac net.HardwareAddr, ip net.IP) error {
	netif := i.ps.netif
	rtMsgBytes, msg := newNeighRtMsg()
	// msg.hdr.Index = uint16(netif.Index)

	// // TODO(paultag): Expose some knobs to allow for expiring routes.
	// msg.hdr.Rmx = unix.RtMetrics{
	// 	Expire: time.Now().Add(time.Second * 10).Unix(),
	// }

	idx := 0
	msg.hdr.Addrs |= unix.RTA_DST
	ip6 := rawSockaddrInet6(ip.To16())
	// ip6.Scope_id = uint32(netif.Index)
	*(*unix.RawSockaddrInet6)(unsafe.Pointer(&msg.data[idx])) = ip6
	idx += int(ip6.Len + (ip6.Len % 8))

	dl := rawSockaddrDatalink(mac)
	dl.Index = uint16(netif.Index)
	msg.hdr.Addrs |= unix.RTA_GATEWAY
	*(*unix.RawSockaddrDatalink)(unsafe.Pointer(&msg.data[idx])) = dl
	idx += int(dl.Len + (dl.Len % 8))

	msg.hdr.Msglen = uint16(unix.SizeofRtMsghdr + idx)
	msg.hdr.Hdrlen = uint16(unix.SizeofRtMsghdr)

	file, err := afRoute()
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(rtMsgBytes[:msg.hdr.Msglen])
	return err
}

// AddNeighbor will add an entry into the ARP / NDP table
func (i Interface) AddNeighbor(mac net.HardwareAddr, ip net.IP) error {
	if ip4 := ip.To4(); len(ip4) == net.IPv4len {
		return i.addNeighborIPv4(mac, ip4)
	}
	return i.addNeighborIPv6(mac, ip)
}

// vim: foldmethod=marker
