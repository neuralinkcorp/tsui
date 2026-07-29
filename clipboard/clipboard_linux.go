package clipboard

/*
#cgo LDFLAGS: -lX11
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

int writeString(
	unsigned char* buf,
	size_t         n,
	uintptr_t      statusHandle
);
*/
import "C"

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"runtime/cgo"
	"sync"
	"time"
	"unsafe"
)

var lock = sync.Mutex{}

func WriteString(str string) error {
	lock.Lock()
	defer lock.Unlock()

	return writeLinuxString(str, os.Getenv("WAYLAND_DISPLAY"), exec.LookPath, writeWaylandString, writeX11String)
}

func writeLinuxString(
	str string,
	waylandDisplay string,
	lookPath func(string) (string, error),
	writeWayland func(string) error,
	writeX11 func(string) error,
) error {
	if waylandDisplay != "" {
		if _, err := lookPath("wl-copy"); err == nil {
			if err := writeWayland(str); err == nil {
				return nil
			}
		}
	}

	return writeX11(str)
}

func writeWaylandString(str string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wl-copy")
	cmd.Stdin = bytes.NewBufferString(str)
	if err := cmd.Run(); err != nil {
		return errUnavailable
	}

	return nil
}

func writeX11String(str string) error {
	buf := []byte(str)

	status := make(chan int)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		statusHandle := cgo.NewHandle(status)

		if len(buf) == 0 {
			C.writeString(nil, 0, C.uintptr_t(statusHandle))
		} else {
			C.writeString((*C.uchar)(unsafe.Pointer(&(buf[0]))), C.size_t(len(buf)), C.uintptr_t(statusHandle))
		}
	}()

	if <-status != 0 {
		return errUnavailable
	}

	return nil
}

// Called from C to update the status channel. Must only be called once per handle,
// because it deletes the handle.
//
//export sendStatus
func sendStatus(statusHandle C.uintptr_t, value C.int) {
	status := cgo.Handle(statusHandle).Value().(chan int)
	status <- int(value)
	cgo.Handle(statusHandle).Delete()
}
