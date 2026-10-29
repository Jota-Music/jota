import { afterAll, beforeEach, expect, mock, test } from "bun:test";
import * as bindings from "@bindings/app";

type Call = { lastModified: number; trust: boolean };

const calls: Call[] = [];
const commits: string[] = [];
const problems: string[] = [];
const saved: string[] = [];
const order: string[] = [];
const files = new Map<string, string>();
let meta: unknown = { lastModified: 1, size: 0 };
// Whether the merge the backend ran left something to write back. The document
// itself never crosses the bridge, so the answer is all the page sees.
let news = false;
// Answers the backend hands back, in order, before falling back to news: the
// second read of a stale copy is what decides the write.
let answers: boolean[] = [];
// Whether the last pass read a copy older than our own write.
let stale = false;
// Whether a pass failed: the panel must keep saying so until one succeeds.
let problem = "";
// Whether the backend holds records the document does not carry yet, and whether
// the snapshots it carries have aged enough to be downloaded again.
let pending = false;
let snapshots = false;
// How many settled passes the backend recorded, and which provider stamp this
// device already merged.
let touched = 0;
let readStamp = 0;
// When the last pass ran, as the backend records it: it survives a reload.
let stamp = 0;
let writes = 0;
// The interval the settings panel promises, in minutes: the launch test needs a
// short one so the grace period is what decides.
let interval = 15;
let uri = "content://doc";
let reloads = 0;
// The picker answers a folder: the native side reports the document inside it
// and whether it had to create it, because an empty one cannot be read yet.
let created = false;
let picked: string | null = null;
// The answer of both cloud pickers: a document, and whether it was just made.
let chosen: string | null = null;
const stages: string[] = [];
const notices: string[] = [];

mock.module("@/lib/shared/i18n", () => ({ t: (key: string) => key }));
mock.module("@/lib/shared/views/stores/errors", () => ({
	logError: () => {},
	notify: (message: string) => notices.push(message),
}));
mock.module("@/lib/shared/views/ui/query-client", () => ({
	queryClient: { invalidateQueries: () => Promise.resolve() },
}));
// The real exports stay: the mock is process-wide, and a partial one makes any
// other test that imports a different binding fail to load.
mock.module("@bindings/app", () => ({
	...bindings,
	LogError: () => Promise.resolve(),
	SetSyncFolder: (next: string) => {
		uri = next;
		saved.push(next);
		return Promise.resolve();
	},
	SyncAuto: () => Promise.resolve(true),
	// Mirrors the backend: it merges the staged copy and says whether the answer
	// must be written back. The freshness rule that decides this lives in Go, so
	// the page only sends the stamp it read and whether it trusts the copy.
	SyncDocument: (lastModified: number, trust: boolean) => {
		calls.push({ lastModified, trust });
		return Promise.resolve(answers.length ? (answers.shift() ?? false) : news);
	},
	SyncFolder: () => Promise.resolve(uri),
	SyncInterval: () => Promise.resolve(interval),
	SyncOverwrite: () => Promise.resolve(),
	SyncPlaylists: () => Promise.resolve(),
	SyncPending: () => Promise.resolve(pending),
	SyncSnapshotsDue: () => Promise.resolve(snapshots),
	SyncTouched: () => {
		touched++;
		return Promise.resolve();
	},
	SyncProblem: (reason: string) => {
		problems.push(reason);
		return Promise.resolve();
	},
	SyncReload: () => {
		reloads++;
		order.push("reload");
		return Promise.resolve({
			bytes: 0,
			inFile: 0,
			playlists: 1,
			follows: 0,
			saved: 0,
			order: 0,
			settings: 0,
			sessions: 0,
			modified: false,
		});
	},
	SyncSave: () => Promise.resolve(),
	SyncStatus: () =>
		Promise.resolve({
			at: stamp,
			uri,
			stale,
			problem,
			readStamp,
		}),
	SyncWritten: () => {
		writes++;
		return Promise.resolve();
	},
}));

let stageFailure: string | null = null;
let commitFailure: string | null = null;

const answer = (id: string, result: string | null, error: string | null) =>
	queueMicrotask(() => window.__jotaSyncCallback?.(id, result, error));

// The document never crosses the bridge: a pass stages the provider's copy as a
// file, Go merges it, and the answer is committed from the same place.
const bridge = {
	pickSyncFolder: () => picked ?? JSON.stringify({ uri, created }),
	pickSyncDocument: () => chosen ?? JSON.stringify({ uri, created }),
	createSyncDocument: () => chosen ?? JSON.stringify({ uri, created }),
	stageSyncText: (id: string, path: string) => {
		stages.push(path);
		if (stageFailure) return answer(id, null, stageFailure);
		if (!files.has(path)) return answer(id, null, "unreadable");
		answer(id, JSON.stringify(meta), null);
	},
	commitSyncText: (id: string, path: string) => {
		if (commitFailure) return answer(id, null, commitFailure);
		order.push("commit");
		commits.push(path);
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

const { chooseFolder, openDocument, createDocument, autosave, reload } =
	await import("@/lib/music/app/backup");

// The window stub lives only for this file: other modules guard on
// `typeof window` and would then reach for a document that does not exist here.
afterAll(() => Reflect.deleteProperty(globalThis, "window"));

beforeEach(() => {
	calls.length = 0;
	commits.length = 0;
	problems.length = 0;
	saved.length = 0;
	stageFailure = null;
	commitFailure = null;
	news = false;
	answers = [];
	stale = false;
	problem = "";
	pending = false;
	snapshots = false;
	touched = 0;
	readStamp = 0;
	stamp = 0;
	writes = 0;
	reloads = 0;
	created = false;
	picked = null;
	chosen = null;
	stages.length = 0;
	notices.length = 0;
	order.length = 0;
});

// The folder's document is opened before it is stored: an unreadable one would
// fail every later pass, and the stage is what proves it readable.
test("a folder with a document stages it and saves it", async () => {
	uri = "content://ok";
	meta = { lastModified: 1, size: 9 };
	files.set(uri, "remote-v1");

	expect(await chooseFolder()).toBe(uri);
	expect(stages).toEqual([uri]);
	expect(saved).toEqual([uri]);
});

// A provider that answers no cursor row still gives us a readable document, so
// the choice is kept: rejecting it would leave the user with no way to sync.
test("a document without metadata is still saved", async () => {
	uri = "content://nodata";
	meta = {};
	files.set(uri, "remote-v1");

	expect(await chooseFolder()).toBe(uri);
	expect(saved).toEqual([uri]);
});

// The document the native side just created is empty and some providers cannot
// open it yet, so it is saved untouched and the first pass fills it.
test("a folder without a document is saved without staging it", async () => {
	uri = "content://created";
	created = true;

	expect(await chooseFolder()).toBe(uri);
	expect(stages).toEqual([]);
	expect(saved).toEqual([uri]);
});

test("a folder whose document cannot be read is not saved", async () => {
	uri = "content://unreadable";
	meta = { lastModified: 1, size: 9 };

	expect(chooseFolder()).rejects.toThrow("settings.autosave.errors.read");
	expect(saved).toEqual([]);
});

// A folder the native side could not prepare (a tree grant that cannot create
// the document in it) is a write failure, not a pick the user made.
test("a native failure while preparing the folder is not saved", async () => {
	picked = JSON.stringify({ error: "EACCES: permission denied" });

	expect(chooseFolder()).rejects.toThrow("settings.autosave.errors.write");
	expect(saved).toEqual([]);
});

// Cancelling is not a failure: the panel keeps the location it had and nothing
// is stored. All three pickers answer the same empty string.
test("cancelling a picker keeps the location", async () => {
	picked = "";
	chosen = "";

	expect(await chooseFolder()).toBe("");
	expect(await openDocument()).toBe("");
	expect(await createDocument()).toBe("");
	expect(saved).toEqual([]);
	expect(stages).toEqual([]);
});

// A document opened in the cloud is read before it is stored, exactly like the
// one a folder already holds.
test("a cloud document is staged and saved", async () => {
	uri = "content://drive";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1, size: 9, displayName: "jota.backup.json" };

	expect(await openDocument()).toBe(uri);
	expect(stages).toEqual([uri]);
	expect(saved).toEqual([uri]);
});

// The created document is born empty, so there is nothing to stage and the
// first pass fills it.
test("a created cloud document is saved without staging it", async () => {
	uri = "content://new";
	created = true;
	meta = { lastModified: 1, size: 0, displayName: "jota.backup.json" };

	expect(await createDocument()).toBe(uri);
	expect(stages).toEqual([]);
	expect(saved).toEqual([uri]);
	expect(notices).toEqual([]);
});

// A provider that already held the document answers a duplicate name instead of
// overwriting it, and there is no way to list the folder to check beforehand.
test("a duplicate name is reported", async () => {
	uri = "content://dup";
	created = true;
	meta = { lastModified: 1, size: 0, displayName: "jota.backup.json (1)" };

	expect(await createDocument()).toBe(uri);
	expect(notices).toEqual(["settings.autosave.duplicate"]);
});

// A document read here whose merge produced local news is committed back to the
// provider.
test("a fresh copy with news is written back", async () => {
	uri = "content://fresh";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	news = true;

	await autosave();

	expect(calls).toEqual([{ lastModified: 1000, trust: true }]);
	expect(commits).toEqual([uri]);
});

// Nothing changed, so the answer is the document already on disk: committing it
// again would wake every file watcher on every interval.
test("an unchanged document is not rewritten", async () => {
	uri = "content://same";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	news = true;
	await autosave();

	// One stamp: the pass copies whatever the provider reported, so comparing
	// against a second Date.now() makes the test fail when the clock ticks.
	const at = Date.now() + 1000;
	meta = { lastModified: at, size: 9 };
	news = false;
	await autosave();

	expect(calls.at(-1)).toEqual({
		lastModified: at,
		trust: true,
	});
	expect(commits).toEqual([uri]);
});

// A pass the backend answered with "nothing to write" must not block the next
// one: the phone still pushes what it makes afterwards.
test("a pull-only sync does not block the next write", async () => {
	uri = "content://pullonly";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	news = false;
	await autosave();

	expect(commits).toEqual([]);

	files.set(uri, "remote-v2");
	news = true;
	await autosave();

	expect(commits).toEqual([uri]);
});

// The native calls run off the JS thread, so the promise resolves only when the
// callback answers. A write failure comes back as a platform message, which the
// user never sees: they get a translatable sentence, recorded in the backend so
// the panel can say the pass failed.
test("a native write resolves and a failure rejects with a translated error", async () => {
	uri = "content://bridge";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	news = true;

	expect(await autosave()).toBeUndefined();
	expect(commits).toEqual([uri]);
	expect(writes).toBe(1);

	commitFailure = "EACCES: permission denied";
	news = true;

	expect(autosave()).rejects.toThrow("settings.autosave.errors.write");
	expect(problems).toEqual(["settings.autosave.errors.write"]);
	// The baseline moves only when the provider confirmed the bytes, so a failed
	// commit is retried by the next pass instead of wedging this device.
	expect(writes).toBe(1);
});

// A provider that serves a cached copy refuses the write. The page reads it once
// more and writes the union anyway: refusing forever would keep this device's
// records out of the file, which is the bug the retry exists for.
test("a stale copy is read once more and pushed", async () => {
	uri = "content://stale";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	answers = [false, true];
	stale = true;

	await autosave("lifecycle");

	expect(calls).toEqual([
		{ lastModified: 1000, trust: false },
		{ lastModified: 1000, trust: true },
	]);
	expect(commits).toEqual([uri]);
	expect(writes).toBe(1);
});

// Nothing to write is not stale: the second read would only cost the provider a
// round trip on every interval.
test("a settled copy without news is not read twice", async () => {
	uri = "content://settled";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	answers = [false];

	await autosave("lifecycle");

	expect(calls.length).toBe(1);
	expect(commits).toEqual([]);
	expect(writes).toBe(0);
});

// A stage that fails rejects, and the failed pass must release the lock: the
// next call has to run instead of waiting on the promise that already died.
test("a failed stage rejects and releases the next pass", async () => {
	uri = "content://stage";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	stageFailure = "EACCES: permission denied";

	expect(autosave()).rejects.toThrow("settings.autosave.errors.read");

	stageFailure = null;
	news = true;
	await autosave();

	expect(calls).toEqual([{ lastModified: 1000, trust: true }]);
	expect(commits).toEqual([uri]);
});

// A commit that fails has already merged the copy, so the next pass must still
// be able to write it instead of staying behind the rejected one.
test("a failed commit does not hang the next pass", async () => {
	uri = "content://commit";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	news = true;
	commitFailure = "EACCES: permission denied";

	expect(autosave()).rejects.toThrow("settings.autosave.errors.write");

	commitFailure = null;
	await autosave();

	expect(commits).toEqual([uri]);
});

const clock = Date.now;
// Start past any earlier automatic run, so the interval tests below see a fresh
// window.
let now = Date.now() + 86_400_000;

// Opening the app is not a sync: the window takes focus while it is still
// painting, and the backend ticker reconciles a minute later anyway.
test("auto sync leaves the launch to the backend ticker", async () => {
	const parked = now;
	uri = "content://launch";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	stamp = 0;
	calls.length = 0;

	Date.now = () => clock() + 5_000;
	await autosave("auto");
	expect(calls).toEqual([]);

	interval = 1;
	Date.now = () => clock() + 61_000;
	await autosave("auto");
	interval = 15;
	now = parked;
	Date.now = clock;

	expect(calls).toEqual([{ lastModified: 1000, trust: false }]);
});
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

	expect(calls).toEqual([{ lastModified: 1000, trust: false }]);
});

// A pass the backend already ran seconds ago is what the startup loop leaves
// behind: opening the window must not read the document a second time.
test("auto sync skips a pass that just ran", async () => {
	Date.now = () => now;
	uri = "content://startup";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	stamp = now - 1_000;
	await autosave("auto");
	Date.now = clock;

	expect(calls).toEqual([]);
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

	// Only the manual call reconciled, and it trusts the copy it read.
	expect(calls).toEqual([{ lastModified: 1000, trust: true }]);
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

	expect(calls).toEqual([{ lastModified: 1000, trust: false }]);
});

// The panel's reload applies the copy the provider holds, so it stages it first
// and lets Go restore from the file a pass would have merged.
test("reload stages the document and restores from it", async () => {
	uri = "content://reload";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };

	await reload();
	expect(reloads).toBe(1);
});

// Nothing readable means nothing to apply: the restore must not run over a copy
// that was never staged.
test("reload with an unreadable document applies nothing", async () => {
	uri = "content://unreadable";
	stageFailure = "unreadable";

	expect(reload()).rejects.toThrow("settings.autosave.errors.read");
	expect(reloads).toBe(0);
});

// Both touch the staged files, so a reload asked for during a pass waits for it
// instead of reading the copy that pass is about to commit.
test("a reload waits for the pass in flight", async () => {
	uri = "content://busy";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	news = true;

	await Promise.all([autosave(), reload()]);

	expect(order).toEqual(["commit", "reload"]);
});

// A pass is settled only when the provider still holds the very copy this device
// already merged and nothing here is waiting to be written: the stat is then the
// whole pass, with no download, merge or upload. The backend records that the
// pass ran, so the panel's "last save" still moves.
test("an unchanged stamp with nothing pending is a stat instead of a pass", async () => {
	uri = "content://unchanged";
	meta = { lastModified: 1000, size: 9 };
	readStamp = 1000;

	await autosave("lifecycle");

	expect(stages).toEqual([]);
	expect(calls).toEqual([]);
	expect(touched).toBe(1);
});

// The provider served another copy, so the document is downloaded and merged.
test("a moved stamp is downloaded", async () => {
	uri = "content://moved";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1001, size: 9 };
	readStamp = 1000;

	await autosave("lifecycle");

	expect(stages).toEqual([uri]);
	expect(calls).toEqual([{ lastModified: 1001, trust: false }]);
});

// This device holds records the document does not carry, so the pass must reach
// the file even when the provider's copy did not move.
test("local news is downloaded even when the stamp did not move", async () => {
	uri = "content://news";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	readStamp = 1000;
	pending = true;

	await autosave("lifecycle");

	expect(stages).toEqual([uri]);
	expect(calls).toEqual([{ lastModified: 1000, trust: false }]);
});

// A save the user asked for is never answered with a stat.
test("a manual save always downloads", async () => {
	uri = "content://manual";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	readStamp = 1000;

	await autosave("manual");

	expect(stages).toEqual([uri]);
	expect(calls).toEqual([{ lastModified: 1000, trust: true }]);
});

// The snapshots the document carries have aged, so the pass that refreshes them
// has to read the file.
test("due snapshots are downloaded", async () => {
	uri = "content://snapshots";
	files.set(uri, "remote-v1");
	meta = { lastModified: 1000, size: 9 };
	readStamp = 1000;
	snapshots = true;

	await autosave("lifecycle");

	expect(stages).toEqual([uri]);
	expect(calls).toEqual([{ lastModified: 1000, trust: false }]);
});

// A copy the backend refused or a pass that failed is a reason to read again: a
// stat would lose the retry the stale flag and the problem exist for.
test("a stale or failed pass is read again instead of settled", async () => {
	uri = "content://stale";
	meta = { lastModified: 1000, size: 9 };
	readStamp = 1000;
	stale = true;

	await autosave("lifecycle");

	expect(stages).toEqual([uri, uri]);
	expect(calls).toEqual([
		{ lastModified: 1000, trust: false },
		{ lastModified: 1000, trust: true },
	]);

	stages.length = 0;
	calls.length = 0;
	stale = false;
	problem = "disk full";

	await autosave("lifecycle");

	expect(stages).toEqual([uri]);
	expect(calls).toEqual([{ lastModified: 1000, trust: false }]);
});

// A provider that reports no stamp says nothing about which copy this device
// already merged, so it can never be proven current.
test("a document without a stamp is always downloaded", async () => {
	uri = "content://nostamp";
	files.set(uri, "remote-v1");
	meta = { lastModified: 0, size: 9 };
	readStamp = 0;

	await autosave("lifecycle");

	expect(stages).toEqual([uri]);
	expect(calls).toEqual([{ lastModified: 0, trust: false }]);
});
