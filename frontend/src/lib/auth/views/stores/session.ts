import {
	SpotifyCancelLogin,
	SpotifyDisconnect,
	SpotifyGetStatus,
	SpotifyLogin,
	SpotifyLoginAndWait,
	SpotifyReconnect,
} from "@bindings/app";
import { signal } from "@preact/signals";
import { open } from "@/lib/shared/utils/open";

export const spotifyConnected = signal(false);
// A stored account whose session could not be restored: Spotify is unreachable
// but the cached catalog still works, so the UI reads it and says so.
export const spotifyDegraded = signal(false);
export const spotifyUser = signal<string | null>(null);
export const spotifyReady = signal(false);

export async function syncSpotifyStatus(): Promise<void> {
	try {
		const status = await SpotifyGetStatus();
		spotifyConnected.value = status.connected;
		spotifyDegraded.value = status.degraded;
		spotifyUser.value = status.user ?? null;
	} catch {
		spotifyConnected.value = false;
		spotifyDegraded.value = false;
		spotifyUser.value = null;
	} finally {
		spotifyReady.value = true;
	}
}

// Manual retry for the degraded banner. The backend watcher is the safety net;
// this just lets the user not wait out the backoff.
export async function retrySpotify(): Promise<boolean> {
	try {
		if (!(await SpotifyReconnect())) return false;
	} catch {
		return false;
	}
	await syncSpotifyStatus();
	return true;
}

export async function disconnectSpotify(): Promise<void> {
	try {
		await SpotifyDisconnect();
	} catch {}
	await syncSpotifyStatus();
}

export async function loginSpotifyAndWait(): Promise<boolean> {
	try {
		const url = await SpotifyLogin();
		if (!url) return false;

		await open(url);
		const returned = watchReturn();
		try {
			await SpotifyLoginAndWait();
		} finally {
			returned();
		}

		await syncSpotifyStatus();
		return true;
	} catch (err) {
		console.error("[auth] Login error:", err);
		await syncSpotifyStatus();
		return false;
	}
}

// A closed browser tab never reaches the loopback callback, so the pending wait
// would hold the panel until the timeout. This is the button that ends it.
export async function cancelSpotifyLogin(): Promise<void> {
	try {
		await SpotifyCancelLogin();
	} catch {}
}

// Closing the tab is not observable; the window coming back is. Jota is left for
// the browser, so a window that regains the focus while the flow is still
// pending means the tab was left behind: the wait ends instead of spinning.
// The grace period covers a user who comes back to read the browser again.
const settle = 5_000;

function watchReturn(): () => void {
	let away = !document.hasFocus();
	let done = false;
	let timer: number | undefined;

	const poll = window.setInterval(() => {
		if (!document.hasFocus()) {
			away = true;
			return;
		}
		if (!away || done) return;
		done = true;
		window.clearInterval(poll);
		timer = window.setTimeout(() => void cancelSpotifyLogin(), settle);
	}, 500);

	return () => {
		window.clearInterval(poll);
		window.clearTimeout(timer);
	};
}
