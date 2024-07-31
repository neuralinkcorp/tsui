// @see https://developer.apple.com/documentation/appkit/nspasteboard?language=objc

#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

// Synchronously writes `n` bytes from `buf` to the clipboard.
int writeString(const void *buf, NSInteger n) {
	NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
	NSData *data = [NSData dataWithBytes: buf length: n];
	[pasteboard clearContents];
	BOOL ok = [pasteboard setData: data forType:NSPasteboardTypeString];
	if (!ok) {
		return -1;
	}
	return 0;
}
