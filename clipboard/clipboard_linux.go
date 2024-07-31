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
	"runtime"
	"runtime/cgo"
	"sync"
	"unsafe"
)

var lock = sync.Mutex{}

func WriteString(str string) error {
	lock.Lock()
	defer lock.Unlock()

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
