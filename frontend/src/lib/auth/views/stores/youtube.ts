import {
	ClearYouTubeCookies,
	SetYouTubeCookies,
	YouTubeBrowserLogin,
	YouTubeSignedIn,
} from "@bindings/app";
import { signal } from "@preact/signals";

export const youtubeSignedIn = signal(false);

export async function syncYouTubeStatus(): Promise<void> {
	try {
		youtubeSignedIn.value = await YouTubeSignedIn();
	} catch {
		youtubeSignedIn.value = false;
	}
}

// Storing account cookies is what unlocks age-restricted videos and avoids the
// bot check for that account's session. The UI must show the ban risk first.
export async function saveYouTubeCookies(cookies: string): Promise<void> {
	await SetYouTubeCookies(cookies);
	await syncYouTubeStatus();
}

// Opens a window on youtube.com so the user can sign in normally; the page
// hands its cookies back to the app.
export async function loginYouTubeWithBrowser(): Promise<void> {
	try {
		await YouTubeBrowserLogin();
	} finally {
		await syncYouTubeStatus();
	}
}

export async function signOutYouTube(): Promise<void> {
	try {
		await ClearYouTubeCookies();
	} catch {}
	await syncYouTubeStatus();
}
