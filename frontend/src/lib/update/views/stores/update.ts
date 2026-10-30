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
// than asked for again on every return. It is filed under the version that asked,
// so an app that has just updated never paints the release it already runs.
const CHECK_EVERY = 6 * 60 * 60 * 1000;
const CACHE_KEY = "update-check";

type Remembered = { at: number; version: string; info: Update };

function remembered(): Remembered | null {
	try {
		const raw = localStorage.getItem(CACHE_KEY);
		if (!raw) return null;
		const parsed = JSON.parse(raw) as {
			at?: unknown;
			version?: unknown;
			info?: unknown;
		};
		if (typeof parsed.at !== "number" || !parsed.info) return null;
		if (typeof parsed.version !== "string") return null;
		return {
			at: parsed.at,
			version: parsed.version,
			info: parsed.info as Update,
		};
	} catch {
		return null;
	}
}

function remember(info: Update): void {
	try {
		localStorage.setItem(
			CACHE_KEY,
			JSON.stringify({ at: Date.now(), version: version.value, info }),
		);
	} catch {}
}

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
	// An answer asked by another version says nothing about this one: it is
	// dropped and asked again, which is what stops a fresh install from
	// advertising the release it already runs.
	const known = remembered();
	if (
		known &&
		known.version === version.value &&
		Date.now() - known.at < CHECK_EVERY
	) {
		update.value = known.info;
		return Promise.resolve();
	}
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
