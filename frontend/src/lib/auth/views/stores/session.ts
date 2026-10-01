import {
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
		await SpotifyLoginAndWait();

		await syncSpotifyStatus();
		return true;
	} catch (err) {
		console.error("[auth] Login error:", err);
		await syncSpotifyStatus();
		return false;
	}
}
