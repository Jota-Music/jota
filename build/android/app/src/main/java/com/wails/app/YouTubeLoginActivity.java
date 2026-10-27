package com.wails.app;

import android.app.Activity;
import android.content.Intent;
import android.os.Bundle;
import android.util.Log;
import android.webkit.CookieManager;
import android.webkit.JavascriptInterface;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;

/**
 * A WebView on youtube.com so the user can sign in normally on Android, where
 * the app itself is a single WebView that cannot load an external URL.
 *
 * The probe runs after each page load and polls every few seconds: it fetches
 * /feed/playlists with the WebView's cookies, HttpOnly ones included, and reads
 * YouTube's verdict off the answer. Once that says "LOGGED_IN":true, the native
 * side reads CookieManager and the page's VISITOR_DATA and client version, and
 * answers the app with all three so it can adopt the session it just signed.
 *
 * NOTE: this file lives in the Wails Android template tree
 * (build/android/app/src/main/java/com/wails/app), so an update that regenerates
 * those template files can overwrite it. Re-apply it on top of the regenerated
 * template instead of hand-merging.
 */
public class YouTubeLoginActivity extends Activity {
    private static final String TAG = "YouTubeLogin";

    /** Must match the request code MainActivity routes back here. */
    static final int REQUEST = 7014;
    static final String EXTRA_COOKIES = "cookies";
    static final String EXTRA_VISITOR = "visitor";
    static final String EXTRA_VERSION = "version";

    private static final String PROBE = "(function () {"
            + "  var sent = false;"
            + "  var probing = false;"
            + "  function cfg(key) {"
            + "    try {"
            + "      if (!window.ytcfg) return '';"
            + "      if (typeof window.ytcfg.get === 'function')"
            + "        return window.ytcfg.get(key) || '';"
            + "      if (window.ytcfg.data_) return window.ytcfg.data_[key] || '';"
            + "    } catch (e) {}"
            + "    return '';"
            + "  }"
            + "  function send() {"
            + "    if (sent) return;"
            + "    sent = true;"
            + "    var version = cfg('INNERTUBE_CONTEXT_CLIENT_VERSION')"
            + "      || cfg('INNERTUBE_CLIENT_VERSION') || '';"
            + "    JotaYouTube.session(cfg('VISITOR_DATA') || '', version);"
            + "  }"
            + "  function step() {"
            + "    if (sent || probing) return;"
            + "    if (location.hostname !== 'www.youtube.com') return;"
            + "    probing = true;"
            + "    fetch('/feed/playlists', { credentials: 'include' })"
            + "      .then(function (r) { return r.text(); })"
            + "      .then(function (html) {"
            + "        if (html.indexOf('\"LOGGED_IN\":true') !== -1) send();"
            + "      })"
            + "      .catch(function () {})"
            + "      .then(function () { probing = false; });"
            + "  }"
            + "  setInterval(step, 3000);"
            + "  step();"
            + "})();";

    private WebView webView;
    private boolean answered;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        CookieManager cookies = CookieManager.getInstance();
        cookies.setAcceptCookie(true);

        webView = new WebView(this);
        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setDatabaseEnabled(true);
        // keep in sync with webUserAgent in internal/services/youtube/client.go
        settings.setUserAgentString(
                "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36");

        cookies.setAcceptThirdPartyCookies(webView, true);
        webView.addJavascriptInterface(this, "JotaYouTube");
        webView.setWebViewClient(new WebViewClient() {
            @Override
            public void onPageFinished(WebView view, String url) {
                view.evaluateJavascript(PROBE, null);
            }
        });

        setContentView(webView);
        webView.loadUrl("https://www.youtube.com/");
    }

    /**
     * Called by the page once its own playlists answer says the session is signed
     * in, with the visitor id and client version that session speaks as.
     */
    @JavascriptInterface
    public void session(String visitor, String version) {
        runOnUiThread(() -> {
            if (answered) return;
            String raw = CookieManager.getInstance().getCookie("https://www.youtube.com");
            if (raw == null || raw.isEmpty()) return;
            String sessionVisitor = visitor == null ? "" : visitor;
            String sessionVersion = version == null ? "" : version;
            Log.i(TAG, "cookie names: " + names(raw));
            Log.i(TAG, "session visitor " + sessionVisitor.length() + " bytes, client " + sessionVersion);
            answered = true;
            CookieManager.getInstance().flush();
            setResult(RESULT_OK, new Intent()
                    .putExtra(EXTRA_COOKIES, raw)
                    .putExtra(EXTRA_VISITOR, sessionVisitor)
                    .putExtra(EXTRA_VERSION, sessionVersion));
            finish();
        });
    }

    /** The names only, never the values: those are the account. */
    private static String names(String header) {
        StringBuilder out = new StringBuilder();
        for (String part : header.split(";")) {
            String name = part.split("=", 2)[0].trim();
            if (name.isEmpty()) continue;
            if (out.length() > 0) out.append(' ');
            out.append(name);
        }
        return out.toString();
    }

    @Override
    protected void onDestroy() {
        if (webView != null) {
            webView.destroy();
            webView = null;
        }
        super.onDestroy();
    }
}
