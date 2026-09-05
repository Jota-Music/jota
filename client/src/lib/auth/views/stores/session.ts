import { computed, signal } from "@preact/signals";
import { get } from "@/lib/shared/api";

const AUTH_SYNC_BRIDGE_ID = "jota:auth-sync";

const authSyncBridge: BroadcastChannel | null =
	typeof BroadcastChannel === "undefined"
		? null
		: new BroadcastChannel(AUTH_SYNC_BRIDGE_ID);

function propagateAuthChange(): void {
	try {
		authSyncBridge?.postMessage({ kind: "auth/changed" as const });
	} catch {
		// ignore
	}
}

if (authSyncBridge) {
	authSyncBridge.addEventListener("message", () => {
		void syncAuth();
	});
}

export const currentUser = signal<string | null>(null);

export const authKnown = signal(false);

export type AuthPhase = "loading" | "guest" | "signedIn";

export const authPhase = computed<AuthPhase>(() => {
	if (!authKnown.value) return "loading";
	return currentUser.value !== null ? "signedIn" : "guest";
});

interface MeResponse {
	user?: string;
}

export async function syncAuth(): Promise<void> {
	try {
		const url = new URL("/api/auth/me", location.origin);
		const res = await fetch(url, {
			method: "GET",
			credentials: "include",
			headers: { Accept: "application/json" },
		});
		if (!res.ok) {
			currentUser.value = null;
			return;
		}
		const data = (await res.json()) as MeResponse;
		currentUser.value =
			typeof data.user === "string" && data.user.length > 0 ? data.user : null;
	} catch {
		currentUser.value = null;
	} finally {
		authKnown.value = true;
	}
}

export function logIn(userName: string): void {
	currentUser.value = userName;
	authKnown.value = true;
	propagateAuthChange();
}

export async function logOut(): Promise<void> {
	try {
		await fetch("/api/auth/log-out", {
			method: "POST",
			credentials: "include",
		});
	} catch {
		// best-effort
	}
	await syncAuth();
	propagateAuthChange();
}

export const spotifyConnected = signal(false);
export const spotifyUser = signal<string | null>(null);

interface SpotifyStatusResponse {
	connected: boolean;
	user?: string;
}

export async function syncSpotifyStatus(): Promise<void> {
	try {
		const res = await get<SpotifyStatusResponse>("/spotify/status");
		spotifyConnected.value = res.connected;
		spotifyUser.value = res.user ?? null;
	} catch {
		spotifyConnected.value = false;
		spotifyUser.value = null;
	}
}

export async function reconnectSpotify(): Promise<void> {
	try {
		const res = await fetch("/api/spotify/reconnect", {
			method: "POST",
			credentials: "include",
		});
		if (!res.ok) return;
		await syncSpotifyStatus();
	} catch {
		// best-effort
	}
}

export async function disconnectSpotify(): Promise<void> {
	try {
		await fetch("/api/spotify/disconnect", {
			method: "POST",
			credentials: "include",
		});
	} catch {
		// best-effort
	}
	await syncSpotifyStatus();
}

export async function loginSpotify(): Promise<string | null> {
	try {
		const res = await fetch("/api/spotify/login", {
			method: "POST",
			credentials: "include",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ origin: window.location.origin }),
		});
		if (!res.ok) return null;
		const data = (await res.json()) as { url?: string };
		return typeof data.url === "string" ? data.url : null;
	} catch {
		return null;
	}
}

export async function loginSpotifyWithCode(code: string): Promise<boolean> {
	try {
		const res = await fetch("/api/spotify/login/callback", {
			method: "POST",
			credentials: "include",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ code }),
		});
		return res.ok;
	} catch {
		return false;
	}
}
