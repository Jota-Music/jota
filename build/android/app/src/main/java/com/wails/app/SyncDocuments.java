package com.wails.app;

import android.app.Activity;
import android.content.Intent;
import android.net.Uri;
import android.provider.DocumentsContract;
import android.provider.OpenableColumns;
import android.util.Log;

import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.CountDownLatch;

/**
 * SAF document access for the playlist sync folder: pick a document, then read,
 * write and stat it.
 *
 * NOTE: this file lives in the Wails Android template tree
 * (build/android/app/src/main/java/com/wails/app), so an update that regenerates
 * those template files can overwrite it. Keep it small and re-apply it on top of
 * the regenerated template instead of hand-merging.
 */
public class SyncDocuments {
    private static final String TAG = "SyncDocuments";

    /** Must match the request code MainActivity routes back here. */
    static final int PICK_REQUEST = 7013;

    private final Activity activity;
    private volatile CountDownLatch latch;
    private volatile String picked;

    public SyncDocuments(Activity activity) {
        this.activity = activity;
    }

    /** Open the system document picker and return the chosen file URI. */
    public String pick() {
        return launch(new Intent(Intent.ACTION_OPEN_DOCUMENT)
                .addCategory(Intent.CATEGORY_OPENABLE)
                .setType("*/*")
                .putExtra(Intent.EXTRA_MIME_TYPES,
                        new String[]{"application/json", "text/plain", "application/octet-stream"}));
    }

    /**
     * Ask the system to create a new sync document and return its URI, or an
     * empty string when the user cancels.
     */
    public String create() {
        return launch(new Intent(Intent.ACTION_CREATE_DOCUMENT)
                .addCategory(Intent.CATEGORY_OPENABLE)
                .setType("application/json")
                // Must match playlists.BackupFile in the Go backend.
                .putExtra(Intent.EXTRA_TITLE, "jota.backup.json"));
    }

    private String launch(Intent intent) {
        Activity current = activity;
        if (current == null) return "";
        latch = new CountDownLatch(1);
        picked = null;
        Intent request = intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION
                | Intent.FLAG_GRANT_WRITE_URI_PERMISSION
                | Intent.FLAG_GRANT_PERSISTABLE_URI_PERMISSION);
        current.runOnUiThread(() -> current.startActivityForResult(request, PICK_REQUEST));
        try {
            CountDownLatch waiting = latch;
            if (waiting != null) waiting.await();
        } catch (InterruptedException ignored) {
        }
        return picked != null ? picked : "";
    }

    /** Result of the picker, forwarded by MainActivity.onActivityResult. */
    public void picked(int resultCode, Intent data) {
        CountDownLatch waiting = latch;
        try {
            Activity current = activity;
            if (current != null && resultCode == Activity.RESULT_OK
                    && data != null && data.getData() != null) {
                Uri uri = data.getData();
                int granted = data.getFlags()
                        & (Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_GRANT_WRITE_URI_PERMISSION);
                try {
                    current.getContentResolver().takePersistableUriPermission(uri, granted);
                } catch (Exception ignored) {
                }
                picked = uri.toString();
            }
        } catch (Exception e) {
            Log.e(TAG, "picked failed", e);
        }
        if (waiting != null) waiting.countDown();
    }

    /**
     * Read a text file from a content URI. Returns null when it cannot be read.
     * The read goes through an explicit openInputStream: SAF gives no way to
     * force a refresh, so a cloud provider may answer with a cached copy.
     */
    public String read(String uriStr) {
        if (uriStr == null || uriStr.isEmpty()) return null;
        try {
            InputStream is = activity.getContentResolver().openInputStream(Uri.parse(uriStr));
            if (is == null) return null;
            ByteArrayOutputStream baos = new ByteArrayOutputStream();
            byte[] buf = new byte[8192];
            int n;
            while ((n = is.read(buf)) != -1) baos.write(buf, 0, n);
            is.close();
            return baos.toString(StandardCharsets.UTF_8.name());
        } catch (Exception e) {
            Log.e(TAG, "read failed", e);
            return null;
        }
    }

    /** Write text content to a content URI. Returns empty on success. */
    public String write(String uriStr, String content) {
        if (uriStr == null || uriStr.isEmpty()) return "empty uri";
        try {
            OutputStream os = activity.getContentResolver().openOutputStream(Uri.parse(uriStr));
            if (os == null) return "cannot open output stream";
            os.write((content != null ? content : "").getBytes(StandardCharsets.UTF_8));
            os.flush();
            os.close();
            return "";
        } catch (Exception e) {
            Log.e(TAG, "write failed", e);
            return e.getMessage();
        }
    }

    /**
     * Document metadata as {"lastModified":ms,"size":bytes}. A SAF/cloud
     * provider answers from its own cache and offers no way to force a refresh,
     * so callers must treat an old lastModified (or a 0) as "not fresh"
     * instead of rewriting the file on top of it. Returns {} when unknown.
     */
    public String info(String uriStr) {
        if (uriStr == null || uriStr.isEmpty()) return "{}";
        try {
            android.database.Cursor cursor = activity.getContentResolver()
                    .query(Uri.parse(uriStr), null, null, null, null);
            if (cursor == null) return "{}";
            try {
                if (!cursor.moveToFirst()) return "{}";
                long modified = 0;
                long size = -1;
                String displayName = "";
                int modIdx = cursor.getColumnIndex(DocumentsContract.Document.COLUMN_LAST_MODIFIED);
                if (modIdx >= 0 && !cursor.isNull(modIdx)) modified = cursor.getLong(modIdx);
                int sizeIdx = cursor.getColumnIndex(OpenableColumns.SIZE);
                if (sizeIdx >= 0 && !cursor.isNull(sizeIdx)) size = cursor.getLong(sizeIdx);
                int nameIdx = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME);
                if (nameIdx >= 0 && !cursor.isNull(nameIdx)) displayName = cursor.getString(nameIdx);
                return new JSONObject()
                        .put("lastModified", modified)
                        .put("size", size)
                        .put("displayName", displayName)
                        .toString();
            } finally {
                cursor.close();
            }
        } catch (Exception e) {
            Log.e(TAG, "info failed", e);
            return "{}";
        }
    }
}
