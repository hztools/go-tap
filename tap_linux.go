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
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// PlatformOptions are the Linux specific TAP Interface options that we're
// able to use during the TAP inteface creation. This is different per-host,
// and needs to be used carefully.
type PlatformOptions struct {
	// Name will set the name of the TAP interface to override the default
	// of a tap* name (like tap0, tap5, tap3)
	Name string

	// Owner string
	// Group string
}

// ifReqFlags is a very hacky ABI compatable version of ifreq as seen
// in `netdevice(7)`. This is only used to set the interface flags
// on Linux, so we're going to create a stub type to use for that purpose.
type ifReqFlags struct {
	Name  [syscall.IFNAMSIZ]byte
	Flags uint16

	// TODO(paultag): This may break in a nasty nasal demons type of way if
	// the sizeof this struct ever changes meaningfully. This is the size of
	// the struct less the Name, and the uint16.
	_ [40 - syscall.IFNAMSIZ - 2]uint8
}

type platformState struct {
	netif netlink.Link
}

func (ps platformState) Close() error {
	return nil
}

// requestInterface will open the TAP/TUN kernel interface, request a new
// TAP/TUN device, and return the interface's name.
//
// TODO(paultag): eventually we'll want to pass some class of Options
// through, which may be platform specific, since OpenBSD and Linux have
// differnet things we'd want to do with the TAP interface at
// creation time.
func requestInterface(opts Options) ([syscall.IFNAMSIZ]byte, *os.File, platformState, error) {
	var (
		name [syscall.IFNAMSIZ]byte
	)

	if len(name) > syscall.IFNAMSIZ {
		return name, nil, platformState{}, fmt.Errorf("tap: provided interface name is too long")
	}

	fd, err := unix.Open("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return name, nil, platformState{}, err
	}
	file := os.NewFile(uintptr(fd), "/dev/net/tun")
	// Closing the file (or deferring a Close) here will close the TAP
	// interface. We'll go ahead and drop the file.Close function out of this
	// Function, since we don't need anything else.

	var flags uint16 = syscall.IFF_NO_PI
	// TODO(paultag): Here, we need to do a switch if we want a TAP or a TUN
	// interface. Since we don't have good TUN support here, this is going
	// to hardcode to TAP. If you're thinking about patching this, it's
	// going to be a bit more work than just switching this flag and calling
	// it a day :)
	flags |= syscall.IFF_TAP

	// Set the ifreq flags, optionally the name, and ship it using ioctl
	// to create the TAP interface.
	req := ifReqFlags{}
	req.Flags = flags
	copy(req.Name[:], opts.PlatformOptions.Name)
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		file.Fd(),
		syscall.TUNSETIFF,
		uintptr(unsafe.Pointer(&req)),
	)
	if errno != 0 {
		file.Close()
		return name, nil, platformState{}, errno
	}

	netif, err := netlink.LinkByName(unix.ByteSliceToString(req.Name[:]))
	if err != nil {
		file.Close()
		return req.Name, nil, platformState{}, err
	}

	ps := platformState{
		netif: netif,
	}

	// TODO(paultag): Allow for TTUNPERSIST/UNSETOWNER/TUNSETGROUP here.
	return req.Name, file, ps, nil
}

// vim: foldmethod=marker
