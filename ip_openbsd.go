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
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func (i Interface) AddAddr(ip net.IP, network *net.IPNet) error {
	// netintro(4) - SIOCAIFADDR
	network.IP = ip
	if ip4 := ip.To4(); len(ip4) == net.IPv4len {
		return i.addAddrIPv4(ip, network)
	}
	return i.addAddrIPv6(ip, network)
}

// IPv4 Address Support

type ifaliasreqIPv4 struct {
	Name      [syscall.IFNAMSIZ]byte
	Addr      unix.RawSockaddrInet4
	Broadcast unix.RawSockaddrInet4
	Mask      unix.RawSockaddrInet4
}

func rawSockaddrInet4(ip net.IP) unix.RawSockaddrInet4 {
	var ret unix.RawSockaddrInet4
	ret.Family = unix.AF_INET
	ret.Len = unix.SizeofSockaddrInet4
	copy(ret.Addr[:], ip.To4())
	return ret
}

func rawSockaddrDatalink(hw net.HardwareAddr) unix.RawSockaddrDatalink {
	var ret unix.RawSockaddrDatalink
	ret.Len = unix.SizeofSockaddrDatalink
	ret.Family = unix.AF_LINK
	ret.Type = unix.IFT_ETHER
	ret.Nlen = 0 // "name length"; interface name []byte, no trailing \0
	ret.Alen = 6 // "address length"; MAC address
	ret.Slen = 0 // selector length, usually 0
	buf := (*[24]byte)(unsafe.Pointer(&(ret.Data[0])))
	copy(buf[:6], hw)
	return ret
}

func (i Interface) addAddrIPv4(ip net.IP, network *net.IPNet) error {
	file, err := afInet()
	if err != nil {
		return err
	}
	defer file.Close()

	var ifar ifaliasreqIPv4

	ifar.Name = i.name
	ifar.Addr = rawSockaddrInet4(ip)
	ifar.Mask = rawSockaddrInet4(net.IP(network.Mask))

	return ioctl(syscall.SIOCAIFADDR, file.Fd(),
		uintptr(unsafe.Pointer(&ifar)))
}

// IPv6 Address Support

type addrlifetimeIPv6 struct {
	Expire         int64
	Preferred      int64
	ValidLifetime  uint32
	PrefixLifetime uint32
}

type ifaliasreqIPv6 struct {
	Name      [syscall.IFNAMSIZ]byte
	Addr      unix.RawSockaddrInet6
	Broadcast unix.RawSockaddrInet6
	Mask      unix.RawSockaddrInet6
	Flags     uint32
	Lifetime  addrlifetimeIPv6
}

func rawSockaddrInet6(ip net.IP) unix.RawSockaddrInet6 {
	var ret unix.RawSockaddrInet6
	ret.Len = unix.SizeofSockaddrInet6
	ret.Family = unix.AF_INET6
	copy(ret.Addr[:], ip.To16())
	return ret
}

func (i Interface) addAddrIPv6(ip net.IP, network *net.IPNet) error {
	var (
		// TODO: in the future use syscall.SIOCAIFADDR_IN6 when that's a thing.
		SIOCAIFADDR_IN6 uintptr = 0x8080691a
	)

	file, err := afInet6()
	if err != nil {
		return err
	}
	defer file.Close()

	var ifar ifaliasreqIPv6
	ifar.Name = i.name
	ifar.Addr = rawSockaddrInet6(ip)
	ifar.Mask = rawSockaddrInet6(net.IP(network.Mask))
	ifar.Lifetime.ValidLifetime = 0xFFFFFFFF
	ifar.Lifetime.PrefixLifetime = 0xFFFFFFFF

	return ioctl(SIOCAIFADDR_IN6, file.Fd(),
		uintptr(unsafe.Pointer(&ifar)))
}

// vim: foldmethod=marker
