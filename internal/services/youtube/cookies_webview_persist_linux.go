//go:build linux && !android

package youtube

// #cgo pkg-config: webkit2gtk-4.1
// #include <webkit2/webkit2.h>
import "C"

import (
	"log"
	"os"
	"unsafe"
)

// PersistWebviewCookies points WebKitGTK's cookie manager at the store the
// sign-in window reads back: without it the jar stays in memory and the window
// can only hand over the visible half of the session.
func PersistWebviewCookies() {
	dir := webviewDataDir()
	if dir == "" {
		log.Printf("youtube: the sign-in window's cookie store has no data directory")
		return
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Printf("youtube: the sign-in window's cookie store: %v", err)
		return
	}
	path := C.CString(cookieStore())
	defer C.free(unsafe.Pointer(path))
	C.webkit_cookie_manager_set_persistent_storage(
		C.webkit_web_context_get_cookie_manager(C.webkit_web_context_get_default()),
		path,
		C.WEBKIT_COOKIE_PERSISTENT_STORAGE_SQLITE,
	)
}
