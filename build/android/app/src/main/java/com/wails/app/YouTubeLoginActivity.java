package com.wails.app;

import android.annotation.SuppressLint;
import android.content.Intent;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;
import android.webkit.CookieManager;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;

import androidx.appcompat.app.AppCompatActivity;

/**
 * Hosts a WebView so the user can sign in to YouTube normally, then hands the
 * account cookies back to the caller. The app's own WebView is a single
 * fullscreen view owned by MainActivity and cannot load an external URL, so the
 * login gets its own Activity.
 */
public class YouTubeLoginActivity extends AppCompatActivity {
    private static final String TAG = "YouTubeLogin";
    private static final boolean DEBUG = BuildConfig.DEBUG;

    public static final String EXTRA_COOKIES = "cookies";

    private static final String YOUTUBE_URL = "https://www.youtube.com/";
    private static final String USER_AGENT =
            "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) "
                    + "Chrome/131.0.0.0 Mobile Safari/537.36";
    private static final long POLL_MS = 1500;

    private WebView webView;
    private final Handler handler = new Handler(Looper.getMainLooper());
    private boolean finished = false;

    @SuppressLint("SetJavaScriptEnabled")
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setTitle("Sign in to YouTube");

        CookieManager cookieManager = CookieManager.getInstance();
        cookieManager.setAcceptCookie(true);

        webView = new WebView(this);
        setContentView(webView);

        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setUserAgentString(USER_AGENT);
        // Google's sign-in sets cookies across domains.
        cookieManager.setAcceptThirdPartyCookies(webView, true);

        webView.setWebViewClient(new WebViewClient() {
            @Override
            public void onPageFinished(WebView view, String url) {
                super.onPageFinished(view, url);
                check();
            }
        });

        webView.loadUrl(YOUTUBE_URL);
        handler.postDelayed(poll, POLL_MS);
    }

    private final Runnable poll = new Runnable() {
        @Override
        public void run() {
            if (finished || check()) {
                return;
            }
            handler.postDelayed(this, POLL_MS);
        }
    };

    /** Returns true once the signed-in cookies are available and delivered. */
    private boolean check() {
        String cookies = CookieManager.getInstance().getCookie(YOUTUBE_URL);
        if (cookies == null
                || (!cookies.contains("SAPISID=") && !cookies.contains("__Secure-3PAPISID="))) {
            return false;
        }

        finished = true;
        if (DEBUG) {
            Log.d(TAG, "signed in, returning " + cookies.length() + " bytes of cookies");
        }
        Intent result = new Intent();
        result.putExtra(EXTRA_COOKIES, cookies);
        setResult(RESULT_OK, result);
        finish();
        return true;
    }

    @Override
    protected void onDestroy() {
        handler.removeCallbacksAndMessages(null);
        if (webView != null) {
            webView.destroy();
            webView = null;
        }
        super.onDestroy();
    }
}
