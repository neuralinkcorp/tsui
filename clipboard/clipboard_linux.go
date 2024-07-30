package clipboard

/*
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>

int clipboard_write(
	unsigned char* buf,
	size_t         n,
	uintptr_t      handle
);
*/
import "C"

import (
	"runtime"
	"runtime/cgo"
	"sync"
	"unsafe"
)

var (
	lock = sync.Mutex{}
)

func WriteString(buf []byte) error {
	lock.Lock()
	defer lock.Unlock()

	status := make(chan int)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		statusHandle := cgo.NewHandle(status)

		if len(buf) == 0 {
			C.clipboard_write(nil, 0, C.uintptr_t(statusHandle))
		} else {
			C.clipboard_write((*C.uchar)(unsafe.Pointer(&(buf[0]))), C.size_t(len(buf)), C.uintptr_t(statusHandle))
		}
	}()

	if <-status < 0 {
		return errUnavailable
	}

	return nil
}

// Accessed on the C side to update the status channel, because the clipboard operation
// on Linux is asynchronous.
//
//export sync_status
func syncStatus(statusHandle uintptr, value int) {
	status := cgo.Handle(statusHandle).Value().(chan int)
	status <- value
	cgo.Handle(statusHandle).Delete()
}
