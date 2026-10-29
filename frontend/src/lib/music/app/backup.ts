import {
	SetSyncFolder,
	SyncAuto,
	SyncDocument,
	SyncFolder,
	SyncInterval,
	SyncOverwrite,
	SyncPending,
	SyncPlaylists,
	SyncProblem,
	SyncReload,
	SyncSave,
	SyncSnapshotsDue,
	SyncStatus,
	SyncTouched,
	SyncWritten,
} from "@bindings/app";
import { t } from "@/lib/shared/i18n";
import { logError, notify } from "@/lib/shared/views/stores/errors";
import { queryClient } from "@/lib/shared/views/ui/query-client";

const ANDROID = /Android/i.test(navigator.userAgent);

// The launch belongs to the backend's ticker (syncTick in Go, one minute): the
// window gets focus while it is still painting, and reading the document there
// makes the open pay for a pass nobody asked for. Manual calls are not throttled.
const BOOT_GRACE_MS = 60_000;
const bootAt = Date.now();

// Must match playlists.BackupFile in the Go backend.
const BACKUP = "jota.backup.json";

type Info = { lastModified: number; size: number; displayName?: string };

// The document calls run off the JS thread on Android, so each one takes a
// callback id and the native side answers through window.__jotaSyncCallback.
// A synchronous call would freeze the WebView while a provider fetches the file.
// The document itself never crosses the bridge: the native side stages it as a
// file for Go to merge, and writes Go's answer back from the same place.
type Bridge = {
	pickSyncFolder(): string;
	pickSyncDocument(last: string): string;
	createSyncDocument(last: string): string;
	stageSyncText(id: string, uri: string): void;
	commitSyncText(id: string, uri: string): void;
	syncFileInfoAsync(id: string, uri: string): void;
};

type Settle = (result: string | null, error: string | null) => void;

function native(): Bridge | null {
	if (!ANDROID) return null;
	return (window as unknown as { wails?: Bridge }).wails ?? null;
}

export const isAndroid = ANDROID;

declare global {
	interface Window {
		// Set on Android so the native side can request a sync on resume/pause.
		__jotaSync?: () => void;
		// Answers the document calls started by ask().
		__jotaSyncCallback?: (
			id: string,
			result: string | null,
			error: string | null,
		) => void;
	}
}

const waiting = new Map<string, Settle>();
let calls = 0;

window.__jotaSyncCallback = (id, result, error) => {
	const settle = waiting.get(id);
	if (!settle) return;
	waiting.delete(id);
	settle(result, error);
};

// A provider that never answers, or a native call that dies with the process,
// would otherwise leave the promise pending and its entry in waiting forever.
const TIMEOUT = 120_000;

function ask(start: (id: string) => void): Promise<string> {
	return new Promise((resolve, reject) => {
		const id = String(++calls);
		const timer = setTimeout(() => {
			waiting.delete(id);
			reject(new Error("sync call timed out"));
		}, TIMEOUT);
		waiting.set(id, (result, error) => {
			clearTimeout(timer);
			if (error) reject(new Error(error));
			else resolve(result ?? "");
		});
		start(id);
	});
}

// stage copies the provider's copy into the file Go reads and answers its
// metadata, so the provider is asked once and never twice in a pass. A null
// answer is the platform saying the document cannot be read at all, which is
// different from an empty one: the caller answers it with a translated error.
async function stage(bridge: Bridge, uri: string): Promise<Info | null> {
	try {
		const answer = await ask((id) => bridge.stageSyncText(id, uri));
		const parsed = JSON.parse(answer) as Partial<Info>;
		return {
			lastModified:
				typeof parsed.lastModified === "number" ? parsed.lastModified : 0,
			size: typeof parsed.size === "number" ? parsed.size : -1,
			displayName: parsed.displayName,
		};
	} catch (error) {
		logError(String(error), "sync stage");
		return null;
	}
}

// The failure is recorded in the backend, not only raised here: a commit that
// never reached the provider would otherwise leave the panel showing the pass
// before it as the last one.
async function commit(bridge: Bridge, uri: string): Promise<void> {
	// The bridge reports a failure as a raw platform message, which is logged in
	// full and never shown: the user gets a translatable sentence instead.
	try {
		const error = await ask((id) => bridge.commitSyncText(id, uri));
		if (!error) return;
		logError(error, "sync write");
	} catch (error) {
		logError(String(error), "sync write");
	}
	const message = t("settings.autosave.errors.write");
	await SyncProblem(message);
	throw new Error(message);
}

// The picker answers a folder, and the sync document inside it: the URI is the
// document, created by the native side when the folder holds none. A tree grant
// covers the children, not another document by itself.
type Picked = { uri?: string; created?: boolean; error?: string };

// The picker answers a raw platform message on failure, which is logged in full
// and never shown: the user gets a translatable sentence instead. A cancel is an
// empty answer, with no URI and nothing created.
function picked(answer: string): Picked {
	if (!answer) return {};
	let parsed: Picked = {};
	try {
		parsed = JSON.parse(answer) as Picked;
	} catch (error) {
		logError(String(error), "sync pick");
		throw new Error(t("settings.autosave.errors.write"));
	}
	if (parsed.error) {
		logError(parsed.error, "sync pick");
		throw new Error(t("settings.autosave.errors.write"));
	}
	return parsed;
}

// A URI we cannot open would be stored anyway and then fail on every later
// sync, so an existing document is tested before it is saved. The stage is that
// test, and it is also what brings the metadata. A stamp is optional: a document
// without one still syncs, it is simply never trusted as fresh. A document the
// native side just created is skipped: it is empty and some providers cannot
// open it yet, and the pass treats an empty document as one it fills.
//
// A cancelled picker answers nothing, which is not a failure: the panel keeps
// the location it had instead of painting an error.
async function keep(bridge: Bridge, choice: Picked): Promise<string> {
	const uri = choice.uri ?? "";
	if (!uri) return "";
	if (!choice.created) {
		const meta = await stage(bridge, uri);
		if (!meta) throw new Error(t("settings.autosave.errors.read"));
		if (!meta.lastModified) logError("sync file reports no metadata", uri);
	}
	await SetSyncFolder(uri);
	return uri;
}

export async function chooseFolder(): Promise<string> {
	const bridge = native();
	if (!bridge) throw new Error(t("settings.autosave.errors.picker"));
	return keep(bridge, picked(bridge.pickSyncFolder()));
}

// A tree picker cannot reach Drive, whose provider announces no tree support, so
// the cloud is reached one document at a time. The picker starts where the
// current document already lives.
export async function openDocument(): Promise<string> {
	const bridge = native();
	if (!bridge) throw new Error(t("settings.autosave.errors.picker"));
	return keep(bridge, picked(bridge.pickSyncDocument(await SyncFolder())));
}

// The created document is born empty, so the stage that proves an existing one
// readable has nothing to say here: the native answer already says it was just
// made, and the first pass fills it.
//
// A provider that already held the document answers a duplicate name instead of
// overwriting it, and there is no way to list the folder to check beforehand, so
// the name it answered is the only defence there is.
export async function createDocument(): Promise<string> {
	const bridge = native();
	if (!bridge) throw new Error(t("settings.autosave.errors.picker"));
	const uri = await keep(
		bridge,
		picked(bridge.createSyncDocument(await SyncFolder())),
	);
	const name = await syncName(uri);
	if (name && name !== BACKUP) notify(t("settings.autosave.duplicate"));
	return uri;
}

async function info(bridge: Bridge, uri: string): Promise<Info | null> {
	try {
		const answer = await ask((id) => bridge.syncFileInfoAsync(id, uri));
		const parsed = JSON.parse(answer) as Partial<Info>;
		if (typeof parsed.lastModified !== "number") return null;
		return {
			lastModified: parsed.lastModified,
			size: parsed.size ?? -1,
			displayName: parsed.displayName,
		};
	} catch {
		return null;
	}
}

// The document URI is a provider id, not a path; the readable name comes from
// the provider's metadata.
export async function syncName(uri: string): Promise<string> {
	const bridge = native();
	if (!bridge || !uri) return "";
	const found = await info(bridge, uri);
	return found?.displayName ?? "";
}

// Answers whether this pass brought anything into the library, so the caller
// only refetches when the document actually moved. The freshness rule lives in
// the backend, which already holds what we last wrote: a manual save or a forced
// overwrite trusts the copy it just read, an automatic pass is judged against it.
async function merge(
	bridge: Bridge,
	uri: string,
	force: boolean,
	manual: boolean,
): Promise<boolean> {
	// The stage carries the metadata of the very copy it copied, so the freshness
	// rule is judged on the document Go actually merges.
	const meta = await stage(bridge, uri);
	if (!meta) throw new Error(t("settings.autosave.errors.read"));
	let saved = await SyncDocument(meta.lastModified, force || manual);
	// A provider that serves a cached copy refuses the write, because a merge over
	// it could drop what another device has not uploaded yet. Read it once more and
	// write the union anyway: refusing forever would leave this device's records
	// out of the file until the provider feels like refreshing, and every device
	// re-pushes its own records on its next pass.
	if (!saved && (await SyncStatus()).stale) {
		const again = await stage(bridge, uri);
		if (!again) throw new Error(t("settings.autosave.errors.read"));
		saved = await SyncDocument(again.lastModified, true);
	}
	if (!saved) return false;
	await commit(bridge, uri);
	// The baseline moves only here: a commit that failed must not make the next
	// pass believe the provider already holds these bytes.
	await SyncWritten();
	return true;
}

type Mode = "manual" | "auto" | "lifecycle";

let running: Promise<void> | null = null;
let lastAuto = 0;

// Every caller that touches the document goes through here: two at once would
// race for the staged files, so the second one runs after the first instead of
// over it. The slot is the run itself, which is what lets an automatic pass join
// a pass already in flight rather than queue behind it.
function queue<T>(task: () => Promise<T>): Promise<T> {
	const previous = running ?? Promise.resolve();
	const next = previous.then(task, task);
	// The slot is freed before the caller of this task resumes, so a signal that
	// arrives right after a pass joins nothing and starts its own.
	const settle = next.then(clear, clear);
	function clear() {
		if (running === settle) running = null;
	}
	running = settle;
	return next;
}

// The manual save applies the document the location holds to this library, the
// file winning over local deletions because the user asked for it. Android has to
// stage the provider's copy first, so Go finds it where a pass leaves it.
async function reloadFromFile(): Promise<
	Awaited<ReturnType<typeof SyncReload>>
> {
	const bridge = native();
	if (bridge) {
		const uri = await SyncFolder();
		if (!uri) throw new Error(t("settings.autosave.errors.selected"));
		if (!(await stage(bridge, uri)))
			throw new Error(t("settings.autosave.errors.read"));
	}
	const stats = await SyncReload();
	await panel();
	void queryClient.invalidateQueries();
	return stats;
}

export function reload(): Promise<Awaited<ReturnType<typeof SyncReload>>> {
	return queue(reloadFromFile);
}

// A pass is queued unless one is already running: two syncs writing the same
// document at once would race, so a call while one runs waits for that one
// instead. Only a manual call is chained; the lifecycle and timer signals overlap
// by design and are answered by the run already in flight. force is the emergency
// exit offered after a save failed: it writes our document over the file instead
// of trusting the copy the provider serves.
export function autosave(mode: Mode = "manual", force = false): Promise<void> {
	if (running)
		return mode === "manual"
			? running.then(() => autosave("manual", force))
			: running;
	return queue(() => run(mode, force));
}

// Opening the app, leaving it and the periodic pass are the moments the sync
// document is reconciled; the settings switch decides whether that happens.
// Android also calls window.__jotaSync from onResume/onPause, where the native
// side knows the app actually moved and the page cannot miss it.
export function startAutosave(): void {
	window.addEventListener("focus", () => void autosave("auto").catch(() => {}));
	if (!ANDROID) return;
	setInterval(() => void autosave("auto").catch(() => {}), 60_000);
	document.addEventListener(
		"visibilitychange",
		() => void autosave("lifecycle").catch(() => {}),
	);
	window.addEventListener(
		"pagehide",
		() => void autosave("lifecycle").catch(() => {}),
	);
	window.__jotaSync = () => void autosave("lifecycle").catch(() => {});
}

async function run(mode: Mode, force: boolean): Promise<void> {
	if (mode !== "manual" && !(await SyncAuto())) return;
	if (mode === "auto" && !(await due())) return;
	const done = await deliver(mode, force);
	if (done && mode !== "manual") lastAuto = Date.now();
}

async function due(): Promise<boolean> {
	if (Date.now() - bootAt < BOOT_GRACE_MS) return false;
	const minutes = await SyncInterval();
	// Opening the app focuses the window right after the startup pass already
	// pulled the whole document: reading those megabytes back seconds later is
	// pure cost, so a pass this recent is skipped on every platform.
	const at = Math.max(lastAuto, (await SyncStatus()).at ?? 0);
	if (Date.now() - at < 60_000) return false;
	return !ANDROID || Date.now() - lastAuto >= minutes * 60_000;
}

// A pass is settled when the provider still holds the very copy this device
// already merged and nothing here is waiting to be written: statting the
// document is then the whole pass, with no stage, download or commit. Anything
// else (a manual save, a forced overwrite, local or remote movement, a failed
// pass) keeps the full path, because each of those is a reason to read again.
async function settled(
	bridge: Bridge,
	uri: string,
	mode: Mode,
	force: boolean,
): Promise<boolean> {
	if (mode === "manual" || force) return false;
	if (await SyncPending()) return false;
	if (await SyncSnapshotsDue()) return false;
	const status = await SyncStatus();
	if (status.uri !== uri) return false;
	if (status.stale || status.problem) return false;
	if (status.readStamp <= 0) return false;
	const meta = await info(bridge, uri);
	return meta?.lastModified === status.readStamp;
}

async function deliver(mode: Mode, force: boolean): Promise<boolean> {
	const bridge = native();
	if (!bridge) {
		if (force) await SyncOverwrite();
		else await (mode === "manual" ? SyncSave() : SyncPlaylists());
		await panel();
		return true;
	}
	const uri = await SyncFolder();
	if (!uri) return false;
	if (await settled(bridge, uri, mode, force)) {
		await SyncTouched();
		await panel();
		return true;
	}
	const changed = await merge(bridge, uri, force, mode === "manual");
	await panel();
	if (changed) void queryClient.invalidateQueries();
	return true;
}

async function panel(): Promise<void> {
	await queryClient.invalidateQueries({ queryKey: ["sync-status"] });
	await queryClient.invalidateQueries({ queryKey: ["sync-pending"] });
}
