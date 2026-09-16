import {
	SpotifyDisconnect,
	SpotifyGetStatus,
	SpotifyLogin,
	SpotifyLoginAndWait,
} from "@bindings/app";
import { signal } from "@preact/signals";
import { open } from "@/lib/shared/utils/open";

export const spotifyConnected = signal(false);
export const spotifyUser = signal<string | null>(null);
export const spotifyReady = signal(false);

export async function syncSpotifyStatus(): Promise<void> {
	try {
		const status = await SpotifyGetStatus();
		spotifyConnected.value = status.connected;
		spotifyUser.value = status.user ?? null;
	} catch {
		spotifyConnected.value = false;
		spotifyUser.value = null;
	} finally {
		spotifyReady.value = true;
	}
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
