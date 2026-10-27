//go:build !linux || android

package youtube

// PersistWebviewCookies does nothing: WebView2, WKWebView and the Android
// WebView persist their jar on their own.
func PersistWebviewCookies() {}
