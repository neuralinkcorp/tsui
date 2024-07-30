package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Cocoa
#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

int write_string(const void *buf, NSInteger n);
*/
import "C"

import (
	"sync"
	"unsafe"
)

var lock = sync.Mutex{}

func WriteString(buf []byte) error {
	lock.Lock()
	defer lock.Unlock()

	var ok C.int

	if len(buf) == 0 {
		ok = C.write_string(unsafe.Pointer(nil), 0)
	} else {
		ok = C.write_string(unsafe.Pointer(&buf[0]), C.NSInteger(len(buf)))
	}

	if ok != 0 {
		return errUnavailable
	}

	return nil
}
