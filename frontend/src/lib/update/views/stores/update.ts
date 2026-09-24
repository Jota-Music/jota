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
	checking = CheckUpdate()
		.then((info) => {
			update.value = info;
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
