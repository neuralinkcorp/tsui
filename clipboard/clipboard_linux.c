#include <stdlib.h>
#include <stdbool.h>
#include <stdint.h>
#include <string.h>
#include <dlfcn.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

// sendStatus is a function from the Go side. Must only be called once per handle.
extern void sendStatus(uintptr_t statusHandle, int value);

void *libX11;

Display *(*dl_XOpenDisplay)(int);
void (*dl_XCloseDisplay)(Display *);
Window (*dl_XDefaultRootWindow)(Display *);
Window (*dl_XCreateSimpleWindow)(Display *, Window, int, int, int, int, int, int, int);
Atom (*dl_XInternAtom)(Display *, char *, int);
void (*dl_XSetSelectionOwner)(Display *, Atom, Window, unsigned long);
Window (*dl_XGetSelectionOwner)(Display *, Atom);
void (*dl_XNextEvent)(Display *, XEvent *);
int (*dl_XChangeProperty)(Display *, Window, Atom, Atom, int, int, unsigned char *, int);
void (*dl_XSendEvent)(Display *, Window, int, long, XEvent *);

// Returns true if it succeeds, false if it fails.
bool initX11()
{
    if (libX11)
    {
        // Already initialized.
        return true;
    }

    libX11 = dlopen("libX11.so", RTLD_LAZY);
    if (!libX11)
    {
        return false;
    }

    dl_XOpenDisplay = dlsym(libX11, "XOpenDisplay");
    dl_XCloseDisplay = dlsym(libX11, "XCloseDisplay");
    dl_XDefaultRootWindow = dlsym(libX11, "XDefaultRootWindow");
    dl_XCreateSimpleWindow = dlsym(libX11, "XCreateSimpleWindow");
    dl_XInternAtom = dlsym(libX11, "XInternAtom");
    dl_XSetSelectionOwner = dlsym(libX11, "XSetSelectionOwner");
    dl_XGetSelectionOwner = dlsym(libX11, "XGetSelectionOwner");
    dl_XNextEvent = dlsym(libX11, "XNextEvent");
    dl_XChangeProperty = dlsym(libX11, "XChangeProperty");
    dl_XSendEvent = dlsym(libX11, "XSendEvent");

    return true;
}

// Writes `n` bytes from `buf` to the clipboard. Sends an int to the passed `statusHandle`
// (which must be a handle to a Go channel) exactly once: negative for failure, 0 for
// success copying.
//
// X handles clipboards very weirdly and the content only persists as long as the owning
// process continues to run. Therefore, this needs to continue running in a thread until
// the ownership is taken by another app, at which point it will return.
//
// @see https://www.uninformativ.de/blog/postings/2017-04-02/0/POSTING-en.html
void writeString(unsigned char *buf, size_t n, uintptr_t statusHandle)
{
    if (!initX11())
    {
        sendStatus(statusHandle, -1);
        return;
    }

    // Attempt to open an X display.
    Display *display = NULL;
    for (int i = 0; i < 42; i++)
    {
        display = dl_XOpenDisplay(0);
        if (display == NULL)
        {
            continue;
        }
        break;
    }
    if (display == NULL)
    {
        // Couldn't open a display.
        sendStatus(statusHandle, -2);
        return;
    }

    // Create an invisible window to use as an agent for clipboard operations.
    Window window = dl_XCreateSimpleWindow(display, dl_XDefaultRootWindow(display), 0, 0, 1, 1, 0, 0, 0);

    Atom sel = dl_XInternAtom(display, "CLIPBOARD", false);
    Atom utf8 = dl_XInternAtom(display, "UTF8_STRING", false);
    Atom targets_atom = dl_XInternAtom(display, "TARGETS", false);

    // Set our window as the owner of the clipboard.
    dl_XSetSelectionOwner(display, sel, window, CurrentTime);
    if (dl_XGetSelectionOwner(display, sel) != window)
    {
        // We couldn't set ourself as the owner.
        dl_XCloseDisplay(display);
        sendStatus(statusHandle, -3);
        return;
    }

    sendStatus(statusHandle, 0);

    XEvent event;
    while (true)
    {
        dl_XNextEvent(display, &event);

        switch (event.type)
        {
        // We lost ownership of the clipboard, which means we can exit this process.
        case SelectionClear:
            return;

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
                error = dl_XChangeProperty(ev.display, ev.requestor, ev.property,
                                           utf8, 8, PropModeReplace,
                                           buf, n);
            }
            else if (ev.target == targets_atom)
            {
                // Reply with the targets we support (only UTF-8).
                Atom targets[] = {utf8};
                error = dl_XChangeProperty(ev.display, ev.requestor, ev.property,
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
                dl_XSendEvent(display, ev.requestor, 0, 0, (XEvent *)&ev);
            }

            break;
        }
    }
}
