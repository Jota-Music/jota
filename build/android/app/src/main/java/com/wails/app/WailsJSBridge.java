package com.wails.app;

import android.util.Log;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import android.webkit.JavascriptInterface;
import android.webkit.WebView;
import com.wails.app.BuildConfig;

/**
 * WailsJSBridge provides the JavaScript interface that allows the web frontend
 * to communicate with the Go backend. This is exposed to JavaScript as the
 * `window.wails` object.
 *
 * Exposed to the WebView through Android's addJavascriptInterface.
 */
public class WailsJSBridge {
    private static final String TAG = "WailsJSBridge";
    private static final boolean DEBUG = BuildConfig.DEBUG;
    // Pooled threads avoid unbounded thread creation under high call volume.
    private static final ExecutorService executor = Executors.newCachedThreadPool();

    private final WailsBridge bridge;
    private final WebView webView;

    public WailsJSBridge(WailsBridge bridge, WebView webView) {
        this.bridge = bridge;
        this.webView = webView;
    }

    /**
     * Send a message to Go and return the response synchronously.
     * Called from JavaScript: wails.invoke(message)
     *
     * @param message The message to send (JSON string)
     * @return The response from Go (JSON string)
     */
    @JavascriptInterface
    public String invoke(String message) {
        if (DEBUG) Log.d(TAG, "Invoke called: " + message);
        return bridge.handleMessage(message);
    }

    /**
     * Send a message to Go asynchronously.
     * The response will be sent back via a callback.
     * Called from JavaScript: wails.invokeAsync(callbackId, message)
     *
     * @param callbackId The callback ID to use for the response
     * @param message The message to send (JSON string)
     */
    @JavascriptInterface
    public void invokeAsync(final String callbackId, final String payload) {
        if (DEBUG) Log.d(TAG, "InvokeAsync called: " + payload);

        // Handle off the JS thread so we don't block the WebView.
        executor.execute(() -> {
            try {
                String response = bridge.handleRuntimeCall(payload);
                sendCallback(callbackId, response, null);
            } catch (Exception e) {
                Log.e(TAG, "Error in async invoke", e);
                sendCallback(callbackId, null, e.getMessage());
            }
        });
    }

    /**
     * Publish the current playback to the system media notification.
     * Called from JavaScript: wails.mediaUpdate(json)
     *
     * @param json {title, artist, album, artwork, playing, duration, position}
     */
    @JavascriptInterface
    public void mediaUpdate(String json) {
        bridge.updateMediaSession(json);
    }

    /**
     * Remove the media notification. Called from JavaScript: wails.mediaClear()
     */
    @JavascriptInterface
    public void mediaClear() {
        bridge.clearMediaSession();
    }

    /**
     * Hand the downloaded release APK to the system package installer.
     * Called from JavaScript: wails.installApk()
     * The APK must already be staged at "<filesDir>/updates/jota.apk" (the
     * update pipeline moves it there after download + checksum verification).
     */
    @JavascriptInterface
    public void installApk() {
        bridge.installApk();
    }

    /**
     * Log a message from JavaScript to Android's logcat
     * Called from JavaScript: wails.log(level, message)
     *
     * @param level The log level (debug, info, warn, error)
     * @param message The message to log
     */
    @JavascriptInterface
    public void setClipboardText(String text) {
        bridge.setClipboardText(text);
    }

    @JavascriptInterface
    public String getClipboardText() {
        return bridge.getClipboardText();
    }

    /**
     * Take the text of a file another app opened Jota with. The export the app
     * writes is what a device sends itself to carry the YouTube session over.
     * Called from JavaScript: wails.takeIncomingFile()
     */
    @JavascriptInterface
    public String takeIncomingFile() {
        return bridge.takeIncomingFile();
    }

    /**
     * The reason the file another app opened Jota with could not be read.
     * Called from JavaScript: wails.takeIncomingError()
     */
    @JavascriptInterface
    public String takeIncomingError() {
        return bridge.takeIncomingError();
    }

    @JavascriptInterface
    public void log(String level, String message) {
        switch (level.toLowerCase()) {
            case "debug":
                Log.d(TAG + "/JS", message);
                break;
            case "info":
                Log.i(TAG + "/JS", message);
                break;
            case "warn":
                Log.w(TAG + "/JS", message);
                break;
            case "error":
                Log.e(TAG + "/JS", message);
                break;
            default:
                Log.v(TAG + "/JS", message);
                break;
        }
    }

    /**
     * Get the platform name
     * Called from JavaScript: wails.platform()
     *
     * @return "android"
     */
    @JavascriptInterface
    public String platform() {
        return "android";
    }

    /**
     * Check if we're running in debug mode
     * Called from JavaScript: wails.isDebug()
     *
     * @return true if debug build, false otherwise
     */
    @JavascriptInterface
    public boolean isDebug() {
        return BuildConfig.DEBUG;
    }

    /**
     * Send a callback response to JavaScript
     */
    private void sendCallback(String callbackId, String result, String error) {
        if (error != null) {
            send(callbackId, "window._wailsAndroidCallback", null, error);
        } else {
            send(callbackId, "window._wailsAndroidCallback", result != null ? result : "", null);
        }
    }

    /**
     * Answer a sync-file call. The page keeps its own callback table, so the
     * result goes to window.__jotaSyncCallback and not the runtime callback:
     * nothing in the frontend defines that one for these calls.
     */
    private void sendSyncCallback(String callbackId, String result, String error) {
        send(callbackId, "window.__jotaSyncCallback", result, error);
    }

    private void send(String callbackId, String target, String result, String error) {
        final String js = String.format(
                "%s && %s('%s', %s, %s);",
                target,
                target,
                escapeJsString(callbackId),
                jsValue(result),
                jsValue(error)
        );

        webView.post(() -> webView.evaluateJavascript(js, null));
    }

    private String jsValue(String value) {
        return value == null ? "null" : "'" + escapeJsString(value) + "'";
    }

    private String escapeJsString(String str) {
        if (str == null) return "";
        return str.replace("\\", "\\\\")
                .replace("'", "\\'")
                .replace("\n", "\\n")
                .replace("\r", "\\r")
                // JS line terminators (U+2028/U+2029) must be escaped too; built via
                // (char) casts so the Java lexer does not reinterpret them as newlines.
                .replace(String.valueOf((char) 0x2028), "\\u2028")
                .replace(String.valueOf((char) 0x2029), "\\u2029");
    }


    @JavascriptInterface
    public String pickSyncFile() {
        return bridge.pickSyncFile();
    }

    @JavascriptInterface
    public String createSyncFile() {
        return bridge.createSyncFile();
    }

    // A document behind a content provider can take long enough to freeze the
    // WebView while it is read, written or stat-ed, so the page uses these
    // variants: they run on the pool and answer through window.__jotaSyncCallback.

    @JavascriptInterface
    public void readSyncText(final String callbackId, final String uri) {
        executor.execute(() -> {
            String text = bridge.readTextFile(uri);
            if (text == null) sendSyncCallback(callbackId, null, "unreadable document");
            else sendSyncCallback(callbackId, text, null);
        });
    }

    @JavascriptInterface
    public void writeSyncText(final String callbackId, final String uri, final String content) {
        executor.execute(() -> {
            String error = bridge.writeTextFile(uri, content);
            sendSyncCallback(callbackId, null, error == null || error.isEmpty() ? null : error);
        });
    }

    @JavascriptInterface
    public void syncFileInfoAsync(final String callbackId, final String uri) {
        executor.execute(() -> sendSyncCallback(callbackId, bridge.syncFileInfo(uri), null));
    }
}
