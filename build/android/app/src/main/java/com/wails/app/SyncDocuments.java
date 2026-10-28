package com.wails.app;

import android.app.Activity;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.provider.DocumentsContract;
import android.provider.OpenableColumns;
import android.util.Log;

import org.json.JSONObject;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

/**
 * SAF document access for the playlist sync folder: pick a document, stage it
 * into the app files dir for the Go backend, and commit the document back.
 *
 * The document never crosses the bridge as a string: the native side copies it
 * to incoming.json, Go reconciles it into outgoing.json and the copy is written
 * back from there. A library document is megabytes, and a string through the
 * WebView would cost both sides three copies of it.
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

    /** Must match playlists.BackupFile in the Go backend. */
    private static final String BACKUP = "jota.backup.json";

    /** Same staged names as syncIncoming/syncOutgoing in internal/app. */
    private static final String INCOMING = "incoming.json";
    private static final String OUTGOING = "outgoing.json";

    private final Activity activity;
    private volatile CountDownLatch latch;
    private volatile String picked;

    public SyncDocuments(Activity activity) {
        this.activity = activity;
    }

    /**
     * Open the system folder picker and answer the sync document inside the
     * chosen folder, creating it when the folder holds none. An empty string is
     * the user cancelling.
     *
     * The picker is a tree, not a document: it lists every DocumentsProvider
     * (Drive, Dropbox, OneDrive and local storage), and the persistable grant
     * taken over the tree is what keeps the document inside it readable later.
     *
     * The folder is resolved after the picker answers, off the UI thread, since
     * launch() waits on the latch that thread has to count down.
     */
    public String folder() {
        String tree = launch(new Intent(Intent.ACTION_OPEN_DOCUMENT_TREE)
                .addFlags(Intent.FLAG_GRANT_PREFIX_URI_PERMISSION));
        if (tree.isEmpty()) return "";
        return resolve(Uri.parse(tree));
    }

    /**
     * Open the system document picker on an existing backup and answer it. An
     * empty string is the user cancelling.
     *
     * A tree picker cannot reach Drive: its provider announces no tree support,
     * so the cloud is reached one document at a time.
     */
    public String pick(String last) {
        String uri = launch(document(Intent.ACTION_OPEN_DOCUMENT, last));
        if (uri.isEmpty()) return "";
        return answer(uri, false);
    }

    /**
     * Create a backup document through the system picker and answer it. The
     * document is born empty, so there is nothing to stage afterwards.
     */
    public String create(String last) {
        String uri = launch(document(Intent.ACTION_CREATE_DOCUMENT, last)
                .putExtra(Intent.EXTRA_TITLE, BACKUP));
        if (uri.isEmpty()) return "";
        return answer(uri, true);
    }

    private Intent document(String action, String last) {
        Intent intent = new Intent(action)
                .addCategory(Intent.CATEGORY_OPENABLE)
                .setType("application/json");
        if (Build.VERSION.SDK_INT >= 26) {
            String initial = last == null || last.isEmpty() ? cloudRoot() : last;
            if (initial != null) intent.putExtra(DocumentsContract.EXTRA_INITIAL_URI, Uri.parse(initial));
        }
        return intent;
    }

    /**
     * The cloud provider's first writable root, so the picker lands in Drive
     * instead of "Recent". Any failure answers null and the picker falls back to
     * its default location.
     *
     * shortcut: the root is picked by the provider's own flags and labels, which
     * is heuristic; narrow it to a known root id when a provider misbehaves.
     */
    private String cloudRoot() {
        Activity current = activity;
        if (current == null) return null;
        String authority = "com.google.android.apps.docs.storage";
        try {
            android.database.Cursor cursor = current.getContentResolver()
                    .query(DocumentsContract.buildRootsUri(authority),
                            new String[]{DocumentsContract.Root.COLUMN_DOCUMENT_ID,
                                    DocumentsContract.Root.COLUMN_FLAGS},
                            null, null, null);
            if (cursor == null) return null;
            try {
                while (cursor.moveToNext()) {
                    long flags = cursor.getLong(1);
                    if ((flags & DocumentsContract.Root.FLAG_SUPPORTS_CREATE) == 0) continue;
                    String id = cursor.getString(0);
                    if (id == null || id.isEmpty()) continue;
                    return DocumentsContract.buildDocumentUri(authority, id).toString();
                }
            } finally {
                cursor.close();
            }
        } catch (Exception e) {
            Log.w(TAG, "no cloud root", e);
        }
        return null;
    }

    private String resolve(Uri tree) {
        Activity current = activity;
        if (current == null) return error("no activity");
        try {
            String treeId = DocumentsContract.getTreeDocumentId(tree);
            Uri parent = DocumentsContract.buildDocumentUriUsingTree(tree, treeId);
            Uri existing = child(current, tree, treeId);
            if (existing != null) return answer(existing.toString(), false);
            Uri created = DocumentsContract.createDocument(
                    current.getContentResolver(), parent, "application/json", BACKUP);
            if (created == null) return error("cannot create " + BACKUP);
            // createDocument may answer a plain document URI, while the grant is
            // held on the tree, so the URI is rebuilt onto it; otherwise the next
            // openInputStream is refused.
            String documentId = DocumentsContract.getDocumentId(created);
            return answer(DocumentsContract.buildDocumentUriUsingTree(tree, documentId).toString(), true);
        } catch (Exception e) {
            Log.e(TAG, "resolve failed", e);
            return error(e.getMessage());
        }
    }

    /** The backup document among the folder's children, or null when absent. */
    private Uri child(Activity current, Uri tree, String parentId) {
        Uri children = DocumentsContract.buildChildDocumentsUriUsingTree(tree, parentId);
        android.database.Cursor cursor = current.getContentResolver().query(children,
                new String[]{DocumentsContract.Document.COLUMN_DOCUMENT_ID,
                        DocumentsContract.Document.COLUMN_DISPLAY_NAME},
                null, null, null);
        if (cursor == null) return null;
        try {
            while (cursor.moveToNext()) {
                String name = cursor.getString(1);
                if (name != null && name.trim().equalsIgnoreCase(BACKUP)) {
                    return DocumentsContract.buildDocumentUriUsingTree(tree, cursor.getString(0));
                }
            }
        } finally {
            cursor.close();
        }
        return null;
    }

    private String answer(String uri, boolean created) {
        try {
            return new JSONObject().put("uri", uri).put("created", created).toString();
        } catch (Exception e) {
            return "{\"uri\":\"\",\"error\":\"cannot encode answer\"}";
        }
    }

    private String error(String message) {
        try {
            return new JSONObject().put("uri", "")
                    .put("error", message == null ? "unknown" : message).toString();
        } catch (Exception e) {
            return "{\"uri\":\"\",\"error\":\"unknown\"}";
        }
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
            // A picker still open when the activity is recreated never answers,
            // and the JS thread waiting on this call must not hang forever.
            if (waiting != null && !waiting.await(300, TimeUnit.SECONDS)) {
                Log.w(TAG, "picker did not answer");
                return "";
            }
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

    /** The dir Go reads and writes as well: app files dir plus /sync. */
    private File stageDir() {
        Activity current = activity;
        if (current == null) return null;
        File dir = new File(current.getFilesDir(), "sync");
        if (dir.isDirectory() || dir.mkdirs()) return dir;
        Log.e(TAG, "cannot create stage dir");
        return null;
    }

    /**
     * Copy the provider's document into incoming.json and answer its metadata
     * as {"lastModified":ms,"size":bytes,"displayName":"..."}. A null answer is
     * an unreadable document, which the caller reports instead of storing a URI
     * that fails on every later pass.
     *
     * The copy is streamed and the metadata comes from the same pass, so a
     * provider behind the SAF cache is asked once and not twice.
     */
    public String stage(String uriStr) {
        if (uriStr == null || uriStr.isEmpty()) return null;
        File dir = stageDir();
        if (dir == null) return null;
        try {
            InputStream is = activity.getContentResolver().openInputStream(Uri.parse(uriStr));
            if (is == null) return null;
            try {
                OutputStream os = new FileOutputStream(new File(dir, INCOMING));
                try {
                    byte[] buf = new byte[64 * 1024];
                    int n;
                    while ((n = is.read(buf)) != -1) os.write(buf, 0, n);
                    os.flush();
                } finally {
                    os.close();
                }
            } finally {
                is.close();
            }
        } catch (Exception e) {
            Log.e(TAG, "stage failed", e);
            return null;
        }
        return info(uriStr);
    }

    /**
     * Write outgoing.json back to the provider and drop the staged copy Go read.
     * Returns empty on success, the platform message otherwise.
     *
     * The stream is opened in "wt" mode so the copy is truncated: a shorter
     * document written over a longer one would otherwise keep the old tail and
     * leave a document that no other device can parse.
     *
     * outgoing.json stays behind on purpose: Go fingerprints the bytes that were
     * just committed (SyncWritten) and removes the file itself, so the write
     * baseline only moves once the provider really holds the document.
     */
    public String commit(String uriStr) {
        if (uriStr == null || uriStr.isEmpty()) return "empty uri";
        File dir = stageDir();
        if (dir == null) return "no stage dir";
        File incoming = new File(dir, INCOMING);
        File outgoing = new File(dir, OUTGOING);
        try {
            OutputStream os = activity.getContentResolver()
                    .openOutputStream(Uri.parse(uriStr), "wt");
            if (os == null) return "cannot open output stream";
            try {
                InputStream is = new FileInputStream(outgoing);
                try {
                    byte[] buf = new byte[64 * 1024];
                    int n;
                    while ((n = is.read(buf)) != -1) os.write(buf, 0, n);
                    os.flush();
                } finally {
                    is.close();
                }
            } finally {
                os.close();
            }
        } catch (Exception e) {
            Log.e(TAG, "commit failed", e);
            // The staged files stay and Go's baseline does not move, so the next
            // pass merges again and writes the document out.
            return e.getMessage();
        }
        incoming.delete();
        return "";
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
