import { afterAll, beforeEach, expect, mock, test } from "bun:test";

type Call = { data: string; fresh: boolean };

const calls: Call[] = [];
const writes: { uri: string; content: string }[] = [];
const saved: string[] = [];
const files = new Map<string, string>();
let meta: unknown = { lastModified: 1, size: 0 };
let merged = (remote: string) => remote;
let uri = "content://doc";

mock.module("@/lib/shared/i18n", () => ({ t: (key: string) => key }));
mock.module("@/lib/shared/views/ui/query-client", () => ({
	queryClient: { invalidateQueries: () => Promise.resolve() },
}));
let status = { at: 0, writeAt: 0, uri: "", hash: 0 };

const fnv = (text: string) => {
	let h = 0x811c9dc5;
	for (const byte of new TextEncoder().encode(text))
		h = Math.imul(h ^ byte, 0x01000193);
	return h >>> 0;
};

mock.module("@bindings/app", () => ({
	LogError: () => Promise.resolve(),
	SetSyncFolder: (next: string) => {
		uri = next;
		saved.push(next);
		return Promise.resolve();
	},
	SyncAuto: () => Promise.resolve(true),
	SyncDocument: (data: string, fresh: boolean) => {
		calls.push({ data, fresh });
		const out = merged(data);
		// Mirrors the backend: only an applied write moves the freshness baseline.
		const wrote = out !== data && fresh;
		status = {
			at: Date.now(),
			writeAt: wrote ? Date.now() : status.writeAt,
			uri,
			hash: wrote ? fnv(out) : status.hash,
		};
		return Promise.resolve(out);
	},
	SyncFolder: () => Promise.resolve(uri),
	SyncInterval: () => Promise.resolve(15),
	SyncOverwrite: () => Promise.resolve(),
	SyncPlaylists: () => Promise.resolve(),
	SyncSave: () => Promise.resolve(),
	SyncStatus: () => Promise.resolve(status),
}));

let failure: string | null = null;

const answer = (id: string, result: string | null, error: string | null) =>
	queueMicrotask(() => window.__jotaSyncCallback?.(id, result, error));

const bridge = {
	pickSyncFile: () => uri,
	createSyncFile: () => uri,
	readSyncText: (id: string, path: string) =>
		answer(id, files.get(path) ?? null, files.has(path) ? null : "unreadable"),
	writeSyncText: (id: string, path: string, content: string) => {
		if (failure) return answer(id, null, failure);
		files.set(path, content);
		writes.push({ uri: path, content });
		answer(id, "", null);
	},
	syncFileInfoAsync: (id: string) => answer(id, JSON.stringify(meta), null),
};
Object.defineProperty(globalThis.navigator, "userAgent", {
	value: "Android 13",
	configurable: true,
});
Object.defineProperty(globalThis, "window", {
	value: { wails: bridge, addEventListener: () => {} },
	configurable: true,
});

const { chooseFile, createFile, autosave } = await import(
	"@/lib/music/app/backup"
);

// The window stub lives only for this file: other modules guard on
// `typeof window` and would then reach for a document that does not exist here.
afterAll(() => Reflect.deleteProperty(globalThis, "window"));

beforeEach(() => {
	calls.length = 0;
	saved.length = 0;
	writes.length = 0;
	failure = null;
	merged = (remote) => remote;
	status = { at: 0, writeAt: 0, uri: "", hash: 0 };
});

// A provider that answers no cursor row still gives us a readable document, so
// the choice is kept: rejecting it would leave the user with no way to sync.
test("a picked document without metadata is still saved", async () => {
	uri = "content://nodata";
	meta = {};
	files.set(uri, "remote-v1");

	expect(await chooseFile()).toBe(uri);
	expect(saved).toEqual([uri]);
});

// The query answer is not proof of readability: the read is the real test.
test("a picked document that cannot be read is not saved", async () => {
	uri = "content://unreadable";
	meta = { lastModified: 1, size: 9 };

	expect(chooseFile()).rejects.toThrow("settings.autosave.errors.read");
	expect(saved).toEqual([]);
});

// A document we just created is empty and some providers refuse to open it
// right away, so the readability check only applies to a picked document.
test("a created document is saved even when it cannot be read", async () => {
	uri = "content://created";

	expect(await createFile()).toBe(uri);
	expect(saved).toEqual([uri]);
});

test("a readable document is saved", async () => {
	uri = "content://ok";
	meta = { lastModified: 1, size: 9 };
	files.set(uri, "remote-v1");

	expect(await chooseFile()).toBe(uri);
	expect(saved).toEqual([uri]);
});

// A file we have never written is always read as fresh, so its news lands here
// and the merged document goes back.
test("a fresh copy with news is written back", async () => {
	uri = "content://fresh";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	merged = () => "merged-v1";

	await autosave();

	expect(calls).toEqual([{ data: "remote-v1", fresh: true }]);
	expect(writes).toEqual([{ uri, content: "merged-v1" }]);
});

// A cloud provider can hand back a copy older than our own last write. It may
// hold unsynced news, so it is merged, but writing on top of it would drop it.
test("a stale copy is merged but never overwritten", async () => {
	uri = "content://stale";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	merged = (remote) => (remote === "remote-v1" ? "merged-v1" : "merged-v2");
	await autosave("lifecycle");

	files.set(uri, "remote-v2");
	meta = { lastModified: 500, size: 9 };
	await autosave("lifecycle");

	expect(calls.at(-1)).toEqual({ data: "remote-v2", fresh: false });
	expect(writes.filter((w) => w.uri === uri)).toEqual([
		{ uri, content: "merged-v1" },
	]);
});

// Nothing changed, so the merged document is the one already on disk: writing it
// again would wake every file watcher on every interval.
test("an unchanged document is not rewritten", async () => {
	uri = "content://same";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	merged = () => "merged-v1";
	await autosave();

	meta = { lastModified: Date.now() + 1000, size: 9 };
	await autosave();

	expect(calls.at(-1)).toEqual({ data: "merged-v1", fresh: true });
	expect(writes.filter((w) => w.uri === uri)).toEqual([
		{ uri, content: "merged-v1" },
	]);
});

// A sync that only pulled changes no baseline: the phone must still be able to
// push the changes it makes afterwards.
test("a pull-only sync does not block the next write", async () => {
	uri = "content://pullonly";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	merged = () => "remote-v1";
	await autosave();

	expect(writes).toEqual([]);

	files.set(uri, "remote-v2");
	merged = () => "merged-v2";
	await autosave();

	expect(writes.filter((w) => w.uri === uri)).toEqual([
		{ uri, content: "merged-v2" },
	]);
});

// The copy on disk is exactly what we wrote, so nothing foreign can be lost:
// local news still goes out even if the provider keeps reporting the old time.
test("local news is pushed on top of our own last write", async () => {
	uri = "content://own";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	merged = () => "merged-v1";
	await autosave();

	meta = { lastModified: 500, size: 9 };
	merged = () => "merged-v2";
	await autosave();

	expect(writes.filter((w) => w.uri === uri).at(-1)).toEqual({
		uri,
		content: "merged-v2",
	});
});

// Without a timestamp, a copy that differs from what we wrote can only hold
// something we have not seen, so it counts as fresh.
test("a timestamp-less copy that changed is fresh", async () => {
	uri = "content://untimed";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	merged = (remote) => `${remote}-merged`;
	await autosave();

	files.set(uri, "remote-v2");
	meta = { lastModified: 0, size: 9 };
	await autosave();

	expect(calls.at(-1)).toEqual({ data: "remote-v2", fresh: true });
	expect(writes.filter((w) => w.uri === uri).at(-1)).toEqual({
		uri,
		content: "remote-v2-merged",
	});
});

// The native calls run off the JS thread, so the promise resolves only when the
// callback answers. A write failure comes back as a platform message, which the
// user never sees: they get a translatable sentence instead.
test("a native write resolves and a failure rejects with a translated error", async () => {
	uri = "content://bridge";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	merged = () => "merged-v1";

	expect(await autosave()).toBeUndefined();
	expect(writes).toEqual([{ uri, content: "merged-v1" }]);

	failure = "EACCES: permission denied";
	merged = () => "merged-v2";

	expect(autosave()).rejects.toThrow("settings.autosave.errors.write");
});

const clock = Date.now;
// Start past any earlier automatic run, so the interval tests below are not
// throttled by the lifecycle pass the stale-copy case just made.
let now = Date.now() + 86_400_000;
// A run that synced nothing must not burn the interval, or a document chosen
// later would sit untouched until the next window.
test("auto sync with no folder does not start the interval", async () => {
	Date.now = () => now;
	uri = "";
	await autosave("auto");

	uri = "content://later";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	await autosave("auto");
	Date.now = clock;

	expect(calls).toEqual([{ data: "remote-v1", fresh: true }]);
});

test("auto sync waits out the interval after a successful run", async () => {
	Date.now = () => now;
	uri = "content://timed";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	await autosave("auto");

	calls.length = 0;
	now += 60_000;
	await autosave("auto");
	Date.now = clock;

	expect(calls).toEqual([]);
});

// An auto run cancelled by the interval resolves at once; a manual call queued
// behind it must still reconcile instead of being answered by that no-op.
test("a manual sync queued behind a throttled auto run still runs", async () => {
	Date.now = () => now;
	uri = "content://queued";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	await autosave("auto");

	calls.length = 0;
	await Promise.all([autosave("auto"), autosave()]);
	Date.now = clock;

	expect(calls).toEqual([{ data: "remote-v1", fresh: true }]);
});

// Opening or leaving the app is not throttled: the interval window belongs to
// the timer, and a real lifecycle change is exactly when a sync is wanted.
test("a lifecycle sync ignores the interval", async () => {
	Date.now = () => now;
	uri = "content://lifecycle";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	await autosave("auto");

	calls.length = 0;
	await autosave("lifecycle");
	Date.now = clock;

	expect(calls).toEqual([{ data: "remote-v1", fresh: true }]);
});
