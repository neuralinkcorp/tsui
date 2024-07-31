package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Cocoa
#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

int writeString(const void *buf, NSInteger n);
*/
import "C"

import (
	"sync"
	"unsafe"
)

var lock = sync.Mutex{}

func WriteString(str string) error {
	lock.Lock()
	defer lock.Unlock()

	buf := []byte(str)

	var ok C.int

	if len(buf) == 0 {
		ok = C.writeString(unsafe.Pointer(nil), 0)
	} else {
		ok = C.writeString(unsafe.Pointer(&buf[0]), C.NSInteger(len(buf)))
	}

	if ok != 0 {
		return errUnavailable
	}

	return nil
}
