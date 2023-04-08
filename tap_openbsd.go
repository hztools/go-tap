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
	"os"
	"syscall"
	"unsafe"
)

type PlatformOptions struct {
}

type platformState struct {
	netif *net.Interface
}

func (ps platformState) Close() error {
	return nil
}

// tuninfo is used by the TUNSIFINFO ioctl to set the TUN state.
type tuninfo struct {
	MTU   uint32
	Type  uint16
	Flags uint16
	Baud  uint32
}

var (
	// ErrNoUsableDevs will be returned if the hz.tools/tap library can't open
	// any of the system TAP files. This error can only happen on OpenBSD.
	ErrNoUsableDevs = fmt.Errorf("tap: all tap devices could not be opened")
)

// requestInterface will attempt to open the /dev/tap* interfaces until
// we have success or run out of files :)
func requestInterface(opts Options) ([syscall.IFNAMSIZ]byte, *os.File, platformState, error) {
	var (
		i    int
		err  error
		fd   *os.File
		name [syscall.IFNAMSIZ]byte
	)

	// Here, we're going to try to open '/dev/tap%d' devices from 0
	// until we get a not found error, attempting to open the RDWR file

	for {
		fd, err = os.OpenFile(fmt.Sprintf("/dev/tap%d", i), os.O_RDWR, 0)
		if err != nil {
			if os.IsNotExist(err) {
				return name, nil, platformState{}, ErrNoUsableDevs
			}
			// otherwise something went wrong, let's skip it and move on.
			i++
			continue
		}
		break
	}

	info := tuninfo{}
	if err := ioctl(TUNGIFINFO, fd.Fd(), uintptr(unsafe.Pointer(&info))); err != nil {
		return name, nil, platformState{}, err
	}

	info.Type = syscall.IFT_ETHER
	info.Flags |= syscall.IFF_RUNNING | syscall.IFF_UP

	// IFT_ETHER - TAP
	// IFT_TUNNEL - TUN

	if err := ioctl(TUNSIFINFO, fd.Fd(), uintptr(unsafe.Pointer(&info))); err != nil {
		return name, nil, platformState{}, err
	}

	// Great, so we have /dev/tap%d; so let's figure out what our name
	// ought to be here
	ifname := fmt.Sprintf("tap%d", i)
	copy(name[:], ifname)

	netif, err := net.InterfaceByName(ifname)
	if err != nil {
		return name, nil, platformState{}, err
	}
	ps := platformState{
		netif: netif,
	}
	return name, fd, ps, nil
}

const (
	// TUNGIFINFO - TUN Get IF INFO
	TUNGIFINFO uintptr = 0x400c745c

	// TUNSIFINFO - TUN Set IF INFO
	TUNSIFINFO uintptr = 0x800c745b
)

// vim: foldmethod=marker
