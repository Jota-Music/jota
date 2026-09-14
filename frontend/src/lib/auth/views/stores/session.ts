import {
	OpenURL,
	SpotifyDisconnect,
	SpotifyGetStatus,
	SpotifyLogin,
	SpotifyLoginAndWait,
} from "@bindings/app";
import { signal } from "@preact/signals";
import { Browser } from "@wailsio/runtime";

export const spotifyConnected = signal(false);
export const spotifyUser = signal<string | null>(null);

export async function syncSpotifyStatus(): Promise<void> {
	try {
		const status = await SpotifyGetStatus();
		spotifyConnected.value = status.connected;
		spotifyUser.value = status.user ?? null;
	} catch {
		spotifyConnected.value = false;
		spotifyUser.value = null;
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
		console.log("[auth] SpotifyLogin returned:", url);
		if (!url) return false;

		if (isAndroid()) {
			await OpenURL(url);
		} else {
			await Browser.OpenURL(url);
		}
		await SpotifyLoginAndWait();

		console.log("[auth] Login complete");
		await syncSpotifyStatus();
		return true;
	} catch (err) {
		console.error("[auth] Login error:", err);
		await syncSpotifyStatus();
		return false;
	}
}

function isAndroid(): boolean {
	return navigator.userAgent.toLowerCase().includes("android");
}
