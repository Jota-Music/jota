package com.wails.app;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.media.MediaMetadata;
import android.media.session.MediaSession;
import android.media.session.PlaybackState;
import android.os.Build;
import android.os.Handler;
import android.os.IBinder;
import android.os.Looper;
import android.util.Log;

import androidx.annotation.Nullable;

import org.json.JSONObject;

import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;

/**
 * Foreground service that publishes the current playback to the system as a
 * media notification (title / artist / cover + transport controls).
 *
 * Android WebView keeps navigator.mediaSession disabled, so the page drives
 * this service through the JS bridge instead, and the notification's transport
 * buttons are forwarded back to the page as "media:action" events.
 */
public class MediaPlaybackService extends Service {
    private static final String TAG = "MediaPlayback";
    private static final String CHANNEL_ID = "media_playback";
    private static final int NOTIFICATION_ID = 0x4A07; // "JO"

    public static final String ACTION_UPDATE = "com.wails.app.MEDIA_UPDATE";
    public static final String ACTION_CLEAR = "com.wails.app.MEDIA_CLEAR";
    private static final String ACTION_PLAY = "com.wails.app.MEDIA_PLAY";
    private static final String ACTION_PAUSE = "com.wails.app.MEDIA_PAUSE";
    private static final String ACTION_NEXT = "com.wails.app.MEDIA_NEXT";
    private static final String ACTION_PREV = "com.wails.app.MEDIA_PREVIOUS";

    /** Receives transport actions so the activity can forward them to JS. */
    public interface Listener {
        void onAction(String action, Long seekMs);
    }

    private static volatile Listener listener;

    public static void setListener(Listener value) {
        listener = value;
    }

    private final Handler main = new Handler(Looper.getMainLooper());
    private MediaSession session;
    private Bitmap artwork;
    private String artworkUrl = "";
    private String pendingArtworkUrl = "";
    private boolean playing = false;
    private String title = "";
    private String artist = "";
    private String album = "";
    private long durationMs = 0;
    private long positionMs = 0;

    @Override
    public void onCreate() {
        super.onCreate();
        session = new MediaSession(this, "jota");
        session.setActive(true);
        session.setCallback(
                new MediaSession.Callback() {
                    @Override
                    public void onPlay() {
                        forward("play", null);
                    }

                    @Override
                    public void onPause() {
                        forward("pause", null);
                    }

                    @Override
                    public void onStop() {
                        forward("stop", null);
                    }

                    @Override
                    public void onSkipToNext() {
                        forward("next", null);
                    }

                    @Override
                    public void onSkipToPrevious() {
                        forward("previous", null);
                    }

                    @Override
                    public void onSeekTo(long pos) {
                        forward("seek", pos);
                    }
                });
        createChannel();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        String action = intent != null ? intent.getAction() : null;

        if (ACTION_CLEAR.equals(action)) {
            stopSelf();
            return START_NOT_STICKY;
        }
        if (ACTION_PLAY.equals(action)) {
            forward("play", null);
            return START_STICKY;
        }
        if (ACTION_PAUSE.equals(action)) {
            forward("pause", null);
            return START_STICKY;
        }
        if (ACTION_NEXT.equals(action)) {
            forward("next", null);
            return START_STICKY;
        }
        if (ACTION_PREV.equals(action)) {
            forward("previous", null);
            return START_STICKY;
        }
        if (intent != null && intent.hasExtra("json")) {
            apply(intent.getStringExtra("json"));
            publish();
            return START_STICKY;
        }

        // Restarted by the OS with no state to restore: don't show a stale card.
        stopSelf();
        return START_NOT_STICKY;
    }

    private void apply(String json) {
        try {
            JSONObject o = new JSONObject(json);
            title = o.optString("title", title);
            artist = o.optString("artist", artist);
            album = o.optString("album", album);
            playing = o.optBoolean("playing", playing);
            durationMs = (long) (o.optDouble("duration", durationMs / 1000.0) * 1000);
            positionMs = (long) (o.optDouble("position", positionMs / 1000.0) * 1000);
            String art = o.optString("artwork", "");
            if (art.isEmpty()) {
                artwork = null;
                artworkUrl = "";
            } else if (!art.equals(artworkUrl)) {
                loadArtwork(art);
            }
        } catch (Exception e) {
            Log.e(TAG, "bad media payload", e);
        }
        updateSession();
    }

    private void updateSession() {
        if (session == null) return;

        MediaMetadata.Builder md =
                new MediaMetadata.Builder()
                        .putString(MediaMetadata.METADATA_KEY_TITLE, title)
                        .putString(MediaMetadata.METADATA_KEY_ARTIST, artist)
                        .putString(MediaMetadata.METADATA_KEY_ALBUM, album)
                        .putLong(MediaMetadata.METADATA_KEY_DURATION, durationMs);
        if (artwork != null) {
            md.putBitmap(MediaMetadata.METADATA_KEY_ALBUM_ART, artwork);
        }
        session.setMetadata(md.build());

        long actions =
                PlaybackState.ACTION_PLAY
                        | PlaybackState.ACTION_PAUSE
                        | PlaybackState.ACTION_PLAY_PAUSE
                        | PlaybackState.ACTION_STOP
                        | PlaybackState.ACTION_SKIP_TO_NEXT
                        | PlaybackState.ACTION_SKIP_TO_PREVIOUS
                        | PlaybackState.ACTION_SEEK_TO;
        PlaybackState state =
                new PlaybackState.Builder()
                        .setActions(actions)
                        .setState(
                                playing ? PlaybackState.STATE_PLAYING : PlaybackState.STATE_PAUSED,
                                positionMs,
                                playing ? 1f : 0f)
                        .build();
        session.setPlaybackState(state);
    }

    private void loadArtwork(final String url) {
        pendingArtworkUrl = url;
        new Thread(
                        () -> {
                            Bitmap bmp = null;
                            try {
                                HttpURLConnection conn =
                                        (HttpURLConnection) new URL(url).openConnection();
                                conn.setConnectTimeout(10000);
                                conn.setReadTimeout(10000);
                                conn.setInstanceFollowRedirects(true);
                                try (InputStream in = conn.getInputStream()) {
                                    bmp = BitmapFactory.decodeStream(in);
                                }
                                conn.disconnect();
                            } catch (Exception e) {
                                Log.e(TAG, "artwork download failed", e);
                            }
                            final Bitmap result = bmp;
                            if (result == null || !url.equals(pendingArtworkUrl)) return;
                            main.post(
                                    () -> {
                                        if (session == null) return;
                                        artwork = result;
                                        artworkUrl = url;
                                        updateSession();
                                        publish();
                                    });
                        })
                .start();
    }

    private void publish() {
        if (session == null) return;
        Notification n = build();
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            startForeground(NOTIFICATION_ID, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PLAYBACK);
        } else {
            startForeground(NOTIFICATION_ID, n);
        }
    }

    private Notification build() {
        Notification.MediaStyle style =
                new Notification.MediaStyle()
                        .setMediaSession(session.getSessionToken())
                        .setShowActionsInCompactView(0, 1, 2);

        Notification.Builder b =
                Build.VERSION.SDK_INT >= Build.VERSION_CODES.O
                        ? new Notification.Builder(this, CHANNEL_ID)
                        : new Notification.Builder(this);

        b.setSmallIcon(android.R.drawable.ic_media_play)
                .setContentTitle(title.isEmpty() ? "Jota" : title)
                .setContentText(artist)
                .setShowWhen(false)
                .setStyle(style)
                .setVisibility(Notification.VISIBILITY_PUBLIC)
                .setOngoing(playing)
                .setContentIntent(launchIntent())
                .addAction(action(android.R.drawable.ic_media_previous, "Previous", ACTION_PREV))
                .addAction(
                        playing
                                ? action(android.R.drawable.ic_media_pause, "Pause", ACTION_PAUSE)
                                : action(android.R.drawable.ic_media_play, "Play", ACTION_PLAY))
                .addAction(action(android.R.drawable.ic_media_next, "Next", ACTION_NEXT));
        if (artwork != null) {
            b.setLargeIcon(artwork);
        }
        return b.build();
    }

    private Notification.Action action(int icon, CharSequence label, String action) {
        Intent i = new Intent(this, MediaPlaybackService.class).setAction(action);
        PendingIntent pi = PendingIntent.getService(this, action.hashCode(), i, piFlags());
        return new Notification.Action.Builder(icon, label, pi).build();
    }

    private PendingIntent launchIntent() {
        Intent launch = getPackageManager().getLaunchIntentForPackage(getPackageName());
        if (launch == null) return null;
        return PendingIntent.getActivity(this, 0, launch, piFlags());
    }

    private int piFlags() {
        int flags = PendingIntent.FLAG_UPDATE_CURRENT;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            flags |= PendingIntent.FLAG_IMMUTABLE;
        }
        return flags;
    }

    private void createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return;
        NotificationChannel ch =
                new NotificationChannel(CHANNEL_ID, "Playback", NotificationManager.IMPORTANCE_LOW);
        ch.setShowBadge(false);
        NotificationManager nm = (NotificationManager) getSystemService(NOTIFICATION_SERVICE);
        if (nm != null) nm.createNotificationChannel(ch);
    }

    private void forward(String action, Long seekMs) {
        Listener l = listener;
        if (l != null) l.onAction(action, seekMs);
    }

    @Override
    public void onDestroy() {
        stopForeground(true);
        if (session != null) {
            session.setActive(false);
            session.release();
            session = null;
        }
        super.onDestroy();
    }

    @Nullable
    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }
}
