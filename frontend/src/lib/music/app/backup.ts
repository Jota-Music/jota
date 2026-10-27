import {
	SetSyncFolder,
	SyncAuto,
	SyncDocument,
	SyncFolder,
	SyncInterval,
	SyncOverwrite,
	SyncPlaylists,
	SyncSave,
	SyncStatus,
} from "@bindings/app";
import { t } from "@/lib/shared/i18n";
import { logError } from "@/lib/shared/views/stores/errors";
import { queryClient } from "@/lib/shared/views/ui/query-client";

const ANDROID = /Android/i.test(navigator.userAgent);

type Info = { lastModified: number; size: number };

// The document calls run off the JS thread on Android, so each one takes a
// callback id and the native side answers through window.__jotaSyncCallback.
// A synchronous call would freeze the WebView while a provider fetches the file.
type Bridge = {
	pickSyncFile(): string;
	createSyncFile(): string;
	readSyncText(id: string, uri: string): void;
	writeSyncText(id: string, uri: string, content: string): void;
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

function ask(start: (id: string) => void): Promise<string> {
	return new Promise((resolve, reject) => {
		const id = String(++calls);
		waiting.set(id, (result, error) => {
			if (error) reject(new Error(error));
			else resolve(result ?? "");
		});
		start(id);
	});
}

// A null result is the platform saying the document cannot be read at all, which
// is different from an empty one: the caller answers it with a translated error.
async function read(bridge: Bridge, uri: string): Promise<string | null> {
	try {
		return await ask((id) => bridge.readSyncText(id, uri));
	} catch (error) {
		logError(String(error), "sync read");
		return null;
	}
}

async function write(
	bridge: Bridge,
	uri: string,
	content: string,
): Promise<void> {
	// The bridge reports a failure as a raw platform message, which is logged in
	// full and never shown: the user gets a translatable sentence instead.
	try {
		const error = await ask((id) => bridge.writeSyncText(id, uri, content));
		if (!error) return;
		logError(error, "sync write");
	} catch (error) {
		logError(String(error), "sync write");
	}
	throw new Error(t("settings.autosave.errors.write"));
}

export async function chooseFile(): Promise<string> {
	const bridge = native();
	if (!bridge) throw new Error(t("settings.autosave.errors.picker"));

	const uri = bridge.pickSyncFile();
	if (!uri) throw new Error(t("settings.autosave.errors.selected"));
	// A URI we cannot read would be stored anyway and then fail on every later
	// sync, so the pick is tested before it is saved. Metadata is optional: not
	// every provider answers a cursor query, and a document without it still
	// syncs fine.
	if ((await read(bridge, uri)) === null)
		throw new Error(t("settings.autosave.errors.read"));
	if (!(await info(bridge, uri)))
		logError("sync file reports no metadata", uri);
	await SetSyncFolder(uri);
	return uri;
}

export async function createFile(): Promise<string> {
	const bridge = native();
	if (!bridge) throw new Error(t("settings.autosave.errors.picker"));

	const uri = bridge.createSyncFile();
	if (!uri) throw new Error(t("settings.autosave.errors.selected"));
	await SetSyncFolder(uri);
	return uri;
}

// FNV-1a 32-bit over the UTF-8 bytes: the same fingerprint the backend records
// for the document it wrote, so both can be compared after a reload.
function hash(text: string): number {
	let h = 0x811c9dc5;
	for (const byte of new TextEncoder().encode(text))
		h = Math.imul(h ^ byte, 0x01000193);
	return h >>> 0;
}

async function info(bridge: Bridge, uri: string): Promise<Info | null> {
	try {
		const answer = await ask((id) => bridge.syncFileInfoAsync(id, uri));
		const parsed = JSON.parse(answer) as Partial<Info>;
		if (typeof parsed.lastModified !== "number") return null;
		return { lastModified: parsed.lastModified, size: parsed.size ?? -1 };
	} catch {
		return null;
	}
}

// A cloud provider can serve a cached copy of the file we wrote minutes ago.
// Writing our merged document on top of it would drop whatever another device
// has not synced yet, so a write is skipped only on positive evidence of that:
// bytes that are not ours and a provider timestamp that did not move after our
// own last write. Our own bytes, or no timestamp at all, carry no such proof; a
// merge is a union anyway, so writing is safe.
function fresh(
	remote: string,
	info: Info | null,
	written: { at: number; hash: number } | null,
): boolean {
	if (!written) return true;
	if (hash(remote) === written.hash) return true;
	if (!info || info.lastModified === 0) return true;
	return info.lastModified > written.at;
}

async function merge(
	bridge: Bridge,
	uri: string,
	force: boolean,
	manual: boolean,
): Promise<void> {
	const remote = await read(bridge, uri);
	if (remote === null) throw new Error(t("settings.autosave.errors.read"));
	// The backend persisted our last write, so the rule survives a reload. A
	// write timestamp of 0 means we never wrote this document.
	const status = await SyncStatus();
	const previous =
		status.uri === uri && status.writeAt > 0
			? { at: status.writeAt, hash: status.hash }
			: null;
	// A manual save is the user asking for the file now, so it trusts the copy
	// it just read even when the provider's timestamp did not move. Automatic
	// passes keep the conservative rule.
	const current =
		force || manual || fresh(remote, await info(bridge, uri), previous);
	const merged = await SyncDocument(remote, current);
	if (merged === remote || !current) return;
	await write(bridge, uri, merged);
}

type Mode = "manual" | "auto" | "lifecycle";

let running: Promise<void> | null = null;
let lastAuto = 0;

// Every caller goes through here: two syncs writing the same document at once
// would race, so a call while one runs waits for that one instead. Only a manual
// call is chained; the lifecycle and timer signals overlap by design and are
// answered by the run already in flight. force is the emergency exit offered
// after a save failed: it writes our document over the file instead of trusting
// the copy the provider serves.
export function autosave(mode: Mode = "manual", force = false): Promise<void> {
	if (running)
		return mode === "manual"
			? running.then(() => autosave("manual", force))
			: running;
	running = run(mode, force).finally(() => {
		running = null;
	});
	return running;
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
	if (!ANDROID) return true;
	const minutes = await SyncInterval();
	return Date.now() - lastAuto >= minutes * 60_000;
}

async function deliver(mode: Mode, force: boolean): Promise<boolean> {
	const bridge = native();
	if (!bridge) {
		if (force) await SyncOverwrite();
		else await (mode === "manual" ? SyncSave() : SyncPlaylists());
		void queryClient.invalidateQueries();
		return true;
	}
	const uri = await SyncFolder();
	if (!uri) return false;
	await merge(bridge, uri, force, mode === "manual");
	void queryClient.invalidateQueries();
	return true;
}
