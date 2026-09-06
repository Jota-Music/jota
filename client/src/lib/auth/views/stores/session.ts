import { signal } from "@preact/signals";
import { BrowserOpenURL } from "@/wailsjs/runtime/runtime";
import {
	SpotifyDisconnect,
	SpotifyGetStatus,
	SpotifyLogin,
	SpotifyLoginAndWait,
	SpotifyReconnect,
} from "@/wailsjs/go/app/App";

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
			await BrowserOpenURL(url);
		}
		console.log("[auth] Calling SpotifyLoginAndWait...");
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
