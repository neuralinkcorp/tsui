#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <dlfcn.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

// sync_status is a function from the Go side.
extern void sync_status(uintptr_t status_handle, int status);

void *libX11;

Display *(*P_XOpenDisplay)(int);
void (*P_XCloseDisplay)(Display *);
Window (*P_XDefaultRootWindow)(Display *);
Window (*P_XCreateSimpleWindow)(Display *, Window, int, int, int, int, int, int, int);
Atom (*P_XInternAtom)(Display *, char *, int);
void (*P_XSetSelectionOwner)(Display *, Atom, Window, unsigned long);
Window (*P_XGetSelectionOwner)(Display *, Atom);
void (*P_XNextEvent)(Display *, XEvent *);
int (*P_XChangeProperty)(Display *, Window, Atom, Atom, int, int, unsigned char *, int);
void (*P_XSendEvent)(Display *, Window, int, long, XEvent *);
int (*P_XGetWindowProperty)(Display *, Window, Atom, long, long, Bool, Atom, Atom *, int *, unsigned long *, unsigned long *, unsigned char **);
void (*P_XFree)(void *);
void (*P_XDeleteProperty)(Display *, Window, Atom);
void (*P_XConvertSelection)(Display *, Atom, Atom, Atom, Window, Time);

int initX11()
{
    if (libX11)
    {
        // Already initialized.
        return 1;
    }

    libX11 = dlopen("libX11.so", RTLD_LAZY);
    if (!libX11)
    {
        return 0;
    }

    P_XOpenDisplay = dlsym(libX11, "XOpenDisplay");
    P_XCloseDisplay = dlsym(libX11, "XCloseDisplay");
    P_XDefaultRootWindow = dlsym(libX11, "XDefaultRootWindow");
    P_XCreateSimpleWindow = dlsym(libX11, "XCreateSimpleWindow");
    P_XInternAtom = dlsym(libX11, "XInternAtom");
    P_XSetSelectionOwner = dlsym(libX11, "XSetSelectionOwner");
    P_XGetSelectionOwner = dlsym(libX11, "XGetSelectionOwner");
    P_XNextEvent = dlsym(libX11, "XNextEvent");
    P_XChangeProperty = dlsym(libX11, "XChangeProperty");
    P_XSendEvent = dlsym(libX11, "XSendEvent");
    P_XGetWindowProperty = dlsym(libX11, "XGetWindowProperty");
    P_XFree = dlsym(libX11, "XFree");
    P_XDeleteProperty = dlsym(libX11, "XDeleteProperty");
    P_XConvertSelection = dlsym(libX11, "XConvertSelection");

    return 1;
}

// Writes `n` bytes from `buf` to the clipboard. Sends an int to the passed `status_handle`
// (which must be a handle to a Go channel) exactly once: negative for failure, 0 for
// success copying.
//
// X handles clipboards very weirdly and the content only persists as long as the owning
// process continues to run. Therefore, this needs to continue running in a thread until
// the ownership is taken by another app, at which point it will return.
//
// @see https://www.uninformativ.de/blog/postings/2017-04-02/0/POSTING-en.html
void clipboard_write(unsigned char *buf, size_t n, uintptr_t status_handle)
{
    if (!initX11())
    {
        sync_status(status_handle, -1);
        return;
    }

    // Attempt to open an X display.
    Display *display = NULL;
    for (int i = 0; i < 42; i++)
    {
        display = P_XOpenDisplay(0);
        if (display == NULL)
        {
            continue;
        }
        break;
    }
    if (display == NULL)
    {
        // Couldn't open a display.
        sync_status(status_handle, -2);
        return;
    }

    // Create an invisible window to use as an agent for clipboard operations.
    Window window = P_XCreateSimpleWindow(display, P_XDefaultRootWindow(display), 0, 0, 1, 1, 0, 0, 0);

    Atom sel = P_XInternAtom(display, "CLIPBOARD", false);
    Atom utf8 = P_XInternAtom(display, "UTF8_STRING", false);
    Atom targets_atom = P_XInternAtom(display, "TARGETS", false);

    // Set our window as the owner of the clipboard.
    P_XSetSelectionOwner(display, sel, window, CurrentTime);
    if (P_XGetSelectionOwner(display, sel) != window)
    {
        // We couldn't set ourself as the owner.
        P_XCloseDisplay(display);
        sync_status(status_handle, -3);
        return;
    }

    sync_status(status_handle, 0);

    XEvent event;
    while (true)
    {
        P_XNextEvent(display, &event);

        switch (event.type)
        {
        // We lost ownership of the clipboard, which means we can exit this process.
        case SelectionClear:
            return 1;

        // Someone wants to paste our data.
        case SelectionRequest:
            XSelectionRequestEvent *xsr = &event.xselectionrequest;

            if (xsr->selection != sel)
            {
                // Not for us.
                break;
            }

            XSelectionEvent ev = {0};
            int error = 0;

            ev.type = SelectionNotify;
            ev.display = xsr->display;
            ev.requestor = xsr->requestor;
            ev.selection = xsr->selection;
            ev.time = xsr->time;
            ev.target = xsr->target;
            ev.property = xsr->property;

            if (ev.target == utf8)
            {
                // Reply with our string.
                error = P_XChangeProperty(ev.display, ev.requestor, ev.property,
                                          utf8, 8, PropModeReplace,
                                          buf, n);
            }
            else if (ev.target == targets_atom)
            {
                // Reply with the targets we support (only UTF-8).
                Atom targets[] = {utf8};
                error = P_XChangeProperty(ev.display, ev.requestor, ev.property,
                                          XA_ATOM, 32, PropModeReplace,
                                          (unsigned char *)&targets, sizeof(targets) / sizeof(Atom));
            }
            else
            {
                // Deny the request.
                ev.property = None;
            }

            // Send the event if XChangeProperty either succeeded or didn't run.
            if ((error & 2) == 0)
            {
                P_XSendEvent(display, ev.requestor, 0, 0, (XEvent *)&ev);
            }

            break;
        }
    }
}
