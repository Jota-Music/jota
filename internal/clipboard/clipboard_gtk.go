//go:build linux && gtk3

package clipboard

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>

extern void jotaClipboardText(char *text);

static void jotaClipboardOnText(GtkClipboard *clipboard, const gchar *text, gpointer data) {
	(void)clipboard;
	(void)data;
	jotaClipboardText((char *)text);
}

static void jotaClipboardRequest(void) {
	gtk_clipboard_request_text(gtk_clipboard_get(GDK_SELECTION_CLIPBOARD), jotaClipboardOnText, NULL);
}
*/
import "C"

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func init() {
	request = func(deliver func(string)) {
		ask(deliver)
		// The dispatching call only schedules the request; the clipboard owner's
		// answer arrives later on the main loop, never as part of this call.
		application.InvokeSync(func() { C.jotaClipboardRequest() })
	}
}

//export jotaClipboardText
func jotaClipboardText(text *C.char) {
	if text == nil {
		deliver("")
		return
	}
	deliver(C.GoString(text))
}
