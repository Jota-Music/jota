import { CheckUpdate, InstallUpdate, Version } from "@bindings/app";
import { signal } from "@preact/signals";
import { Events } from "@wailsio/runtime";
import { addError } from "@/lib/shared/views/stores/errors";

type Update = Awaited<ReturnType<typeof CheckUpdate>>;

export const version = signal("");
export const update = signal<Update | null>(null);
export const dismissed = signal(false);
export const installing = signal(false);
export const progress = signal<{ written: number; total: number } | null>(null);

Events.On("update:progress", (ev) => {
	progress.value = ev.data as { written: number; total: number };
});

// percent is null while the total size is unknown.
export function percent(): number | null {
	const p = progress.value;
	if (!p || p.total <= 0) return null;
	return Math.min(100, Math.round((p.written / p.total) * 100));
}

let checking: Promise<void> | null = null;

// The answer is kept for hours: the window regains focus constantly, the release
// API is rate limited, and a known update is better shown from the last answer
// than asked for again on every return.
const CHECK_EVERY = 6 * 60 * 60 * 1000;
const CACHE_KEY = "update-check";

function remembered(): { at: number; info: Update } | null {
	try {
		const raw = localStorage.getItem(CACHE_KEY);
		if (!raw) return null;
		const parsed = JSON.parse(raw) as { at?: unknown; info?: unknown };
		if (typeof parsed.at !== "number" || !parsed.info) return null;
		return { at: parsed.at, info: parsed.info as Update };
	} catch {
		return null;
	}
}

function remember(info: Update): void {
	try {
		localStorage.setItem(CACHE_KEY, JSON.stringify({ at: Date.now(), info }));
	} catch {}
}

// The last answer paints before the first frame, so an update that was already
// known is on screen without a check.
const known = remembered();
if (known) update.value = known.info;

export function loadVersion(): Promise<void> {
	if (version.value) return Promise.resolve();
	return Version()
		.then((v) => {
			version.value = v;
		})
		.catch(() => {});
}

export function checkUpdate(): Promise<void> {
	if (checking) return checking;
	if (known && Date.now() - known.at < CHECK_EVERY) return Promise.resolve();
	checking = CheckUpdate()
		.then((info) => {
			update.value = info;
			remember(info);
		})
		.catch(() => {})
		.finally(() => {
			checking = null;
		});
	return checking;
}

export async function install(): Promise<void> {
	installing.value = true;
	progress.value = null;
	try {
		await InstallUpdate();
		// Android: GOOS=android installs hand the verified APK to the system
		// package installer from Java (WailsJSBridge.installApk), so after the
		// download+verify resolves the installer takes over.
		native()?.installApk?.();
		installing.value = false;
	} catch (e) {
		installing.value = false;
		progress.value = null;
		addError(e, `update install ${update.value?.latest ?? ""}`.trim());
	}
}

type NativeBridge = { installApk?: () => void };

function native(): NativeBridge | null {
	if (typeof window === "undefined") return null;
	const wails = (window as unknown as { wails?: NativeBridge }).wails;
	return wails && typeof wails.installApk === "function" ? wails : null;
}
