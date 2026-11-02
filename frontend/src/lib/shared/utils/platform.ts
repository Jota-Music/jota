// The page runs as a WebView on Android, where every request is answered by the
// app's own asset handler instead of the platform's HTTP cache.
export const isAndroid = /Android/i.test(navigator.userAgent);
