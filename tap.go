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

package tap

import (
	"context"
	"os"
	"syscall"

	"github.com/mdlayher/ethernet"
	"golang.org/x/sys/unix"
)

type Interface struct {
	name   [syscall.IFNAMSIZ]byte
	fd     *os.File
	ctx    context.Context
	cancel context.CancelFunc
	ps     platformState
}

func (i Interface) Name() string {
	return unix.ByteSliceToString(i.name[:])
}

func (i Interface) Close() error {
	i.cancel()
	return nil
}

type Options struct {
	PlatformOptions PlatformOptions
}

// New will create a new TAP interface.
func New(ctx context.Context, o Options) (*Interface, error) {
	ctx, cancel := context.WithCancel(ctx)

	name, fd, ps, err := requestInterface(o)
	if err != nil {
		return nil, err
	}

	go func() {
		// Here, we'll sit in a goroutine and close the underlying fd when the
		// context is cancelled. This way, all code using the interface can
		// depend on the Context for state, and not worry about a cancelled
		// context without freeing the interface.
		<-ctx.Done()
		fd.Close()
		ps.Close()
	}()

	return &Interface{
		ctx:    ctx,
		cancel: cancel,
		name:   name,
		fd:     fd,
		ps:     ps,
	}, nil
}

// Read will read and parse the next Ethernet Frame from the TAP.
func (i Interface) Read() (*ethernet.Frame, error) {
	var (
		frame = &ethernet.Frame{}
		buf   = make([]byte, 9100)
	)

	if err := i.ctx.Err(); err != nil {
		return nil, err
	}
	n, err := i.fd.Read(buf)
	if err != nil {
		return nil, err
	}
	if err := frame.UnmarshalBinary(buf[:n]); err != nil {
		return nil, err
	}
	return frame, nil
}

// Write will encode the passed ethernet.Frame to the TAP.
func (i Interface) Write(frame *ethernet.Frame) error {
	buf, err := frame.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = i.fd.Write(buf)
	return err
}

func ioctl(action uintptr, fd uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		action,
		arg,
	)
	if errno == 0 {
		return nil
	}
	return errno
}

// vim: foldmethod=marker
