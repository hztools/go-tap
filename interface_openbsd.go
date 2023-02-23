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
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	// IFDESCRSIZE is the max size of the Description field on OpenBSD.
	// Replace this once syscall or x/sys/unix grows IFDESCRSIZE.
	IFDESCRSIZE int = 64
)

type ifreqDescription struct {
	Name        [syscall.IFNAMSIZ]byte
	Description *[IFDESCRSIZE]byte
}

// SetDescription will set the network interface description to the provided
// string.
func (i Interface) SetDescription(descr string) error {
	file, err := afInet()
	if err != nil {
		return err
	}
	defer file.Close()

	d := [IFDESCRSIZE]byte{}
	copy(d[:], descr)

	req := ifreqDescription{
		Name:        i.name,
		Description: &d,
	}
	if err := ioctl(syscall.SIOCSIFDESCR, file.Fd(), uintptr(unsafe.Pointer(&req))); err != nil {
		return err
	}
	return nil
}

type ifreqFlags struct {
	Name  [syscall.IFNAMSIZ]byte
	Flags uint16
}

// SetUp will set the link state to up or down.
func (i Interface) SetUp(updown bool) error {
	file, err := afInet()
	if err != nil {
		return err
	}
	defer file.Close()

	req := ifreqFlags{Name: i.name}
	if err := ioctl(syscall.SIOCGIFFLAGS, file.Fd(), uintptr(unsafe.Pointer(&req))); err != nil {
		return err
	}

	if updown {
		// Bring the iface up; so let's set the bit and let the syscall
		// roll. This is a noop if the interface is already up.
		req.Flags |= syscall.IFF_UP
	} else {
		// Bring the interface down; let's create a bitmask of the flags
		// without the IFF_UP flag, and send that back in. This is a
		// noop if the interface is already down.
		req.Flags &= (^uint16(syscall.IFF_UP))
	}

	if err := ioctl(syscall.SIOCSIFFLAGS, file.Fd(), uintptr(unsafe.Pointer(&req))); err != nil {
		return err
	}
	return nil
}

type ifreqMTU struct {
	Name [syscall.IFNAMSIZ]byte
	MTU  int32
}

// SetMTU will set the MTU for the created TAP interface.
func (i Interface) SetMTU(sz uint) error {
	file, err := afInet()
	if err != nil {
		return err
	}
	defer file.Close()
	req := ifreqMTU{Name: i.name, MTU: int32(sz)}
	return ioctl(syscall.SIOCSIFMTU, file.Fd(), uintptr(unsafe.Pointer(&req)))
}

type ifreqLLADDR struct {
	Name [syscall.IFNAMSIZ]byte
	Addr unix.RawSockaddr
}

// SetHardwareAddr will set the MAC Address (LLADDR) of the underlying
// TAP interface.
func (i Interface) SetHardwareAddr(addr net.HardwareAddr) error {
	file, err := afInet()
	if err != nil {
		return err
	}
	defer file.Close()

	req := ifreqLLADDR{Name: i.name}
	// Awkwardly, this is not a RawSockaddrDatalink, this is a RawSockaddr
	// with a MAC in the payload. Soooo, cool cool. Let's just do that
	// here.
	req.Addr = unix.RawSockaddr{
		Len:    6,
		Family: unix.AF_LINK,
	}

	// Doubly annoying is a RawSockaddr is a []int8 (wat) not a []uint8,
	// so let's cast this away since this is a bug in Go, and the kernel
	// will understand this memory as a []byte, not as a []int. This makes
	// debugging and interface code a lot harder, but it's possible
	// if we escape the typesystem.
	buf := (*[14]byte)(unsafe.Pointer(&(req.Addr.Data[0])))
	copy(buf[:6], addr)

	return ioctl(syscall.SIOCSIFLLADDR, file.Fd(), uintptr(unsafe.Pointer(&req)))
}

type ifreqGroup struct {
	Name     [syscall.IFNAMSIZ]byte
	GroupLen uint64
	Group    [syscall.IFNAMSIZ]byte
}

func newGroupReq(name [syscall.IFNAMSIZ]byte, group string) (*ifreqGroup, error) {
	if group == "" {
		return nil, fmt.Errorf("tap: interface group is empty")
	}

	// TODO: check if the group ends with a digit (not allowed)

	if len(group) >= syscall.IFNAMSIZ {
		return nil, fmt.Errorf("tap: interface group name is too long")
	}

	req := ifreqGroup{Name: name}
	copy(req.Group[:], []byte(group))
	return &req, nil
}

// AddGroup will add the provided group name to the underlying TAP
// interface.
func (i Interface) AddGroup(group string) error {
	file, err := afInet()
	if err != nil {
		return err
	}
	defer file.Close()
	req, err := newGroupReq(i.name, group)
	if err != nil {
		return err
	}
	return ioctl(syscall.SIOCAIFGROUP, file.Fd(), uintptr(unsafe.Pointer(req)))
}

// RemoveGroup will remove the group name from the TAP interface.
func (i Interface) RemoveGroup(group string) error {
	file, err := afInet()
	if err != nil {
		return err
	}
	defer file.Close()
	req, err := newGroupReq(i.name, group)
	if err != nil {
		return err
	}
	return ioctl(syscall.SIOCDIFGROUP, file.Fd(), uintptr(unsafe.Pointer(req)))
}

// vim: foldmethod=marker
