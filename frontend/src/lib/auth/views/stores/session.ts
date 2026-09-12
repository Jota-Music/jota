import {
	SpotifyDisconnect,
	SpotifyGetStatus,
	SpotifyLogin,
	SpotifyLoginAndWait,
	SpotifyReconnect,
} from "@bindings/app";
import { signal } from "@preact/signals";
import { Browser } from "@wailsio/runtime";

export const currentUser = signal<string | null>(null);
export const authKnown = signal(false);

export async function syncAuth(): Promise<void> {
	authKnown.value = true;
}

export async function logOut(): Promise<void> {}

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

export async function reconnectSpotify(): Promise<void> {
	try {
		await SpotifyReconnect();
	} catch {}
	await syncSpotifyStatus();
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
		if (url) {
			await Browser.OpenURL(url);
		}

		// On Android, the OAuth callback is handled via custom scheme intent
		// (jota://callback), so we don't call SpotifyLoginAndWait.
		// The callback is handled by the native Android intent handler.
		const isAndroid = navigator.userAgent.toLowerCase().includes("android");
		if (!isAndroid) {
			console.log("[auth] Calling SpotifyLoginAndWait...");
			await SpotifyLoginAndWait();
		} else {
			console.log("[auth] Android detected, skipping SpotifyLoginAndWait (handled by custom scheme)");
		}

		console.log("[auth] Login complete");
		await syncSpotifyStatus();
		return true;
	} catch (err) {
		console.error("[auth] Login error:", err);
		await syncSpotifyStatus();
		return false;
	}
}
