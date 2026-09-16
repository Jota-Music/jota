import {
	ClearYouTubeCookies,
	SetYouTubeCookies,
	YouTubeBrowserLogin,
	YouTubeSignedIn,
} from "@bindings/app";
import { signal } from "@preact/signals";
import { Events } from "@wailsio/runtime";

export const youtubeSignedIn = signal(false);

// Set when a track fails because it needs a signed-in YouTube session.
export const youtubeSignInSuggested = signal(false);

Events.On("youtube:signin-required", () => {
	if (!youtubeSignedIn.value) youtubeSignInSuggested.value = true;
});

export function dismissYouTubeSignInSuggestion(): void {
	youtubeSignInSuggested.value = false;
}

export async function syncYouTubeStatus(): Promise<void> {
	try {
		youtubeSignedIn.value = await YouTubeSignedIn();
	} catch {
		youtubeSignedIn.value = false;
	}
	if (youtubeSignedIn.value) youtubeSignInSuggested.value = false;
}

// Storing account cookies is what unlocks age-restricted videos and avoids the
// bot check for that account's session. The UI must show the ban risk first.
export async function saveYouTubeCookies(cookies: string): Promise<void> {
	await SetYouTubeCookies(cookies);
	await syncYouTubeStatus();
}

type WailsNative = { youtubeLogin?: () => void };

function nativeBridge(): WailsNative | null {
	const wails = (window as unknown as { wails?: WailsNative }).wails;
	return wails && typeof wails.youtubeLogin === "function" ? wails : null;
}

// Android has no secondary WebView window, so the Java side opens its own
// Activity and hands the cookies back through this global callback.
function androidLogin(native: WailsNative): Promise<void> {
	return new Promise((resolve, reject) => {
		const w = window as unknown as {
			__jotaYouTubeCookies?: (cookies: string | null) => void;
		};
		const cleanup = () => {
			clearTimeout(timer);
			delete w.__jotaYouTubeCookies;
		};
		const timer = setTimeout(
			() => {
				cleanup();
				reject(new Error("Sign-in timed out"));
			},
			5 * 60 * 1000,
		);

		w.__jotaYouTubeCookies = (cookies) => {
			cleanup();
			if (!cookies) {
				reject(new Error("Sign-in cancelled"));
				return;
			}
			saveYouTubeCookies(cookies).then(resolve, reject);
		};

		native.youtubeLogin?.();
	});
}

// Opens a YouTube sign-in surface: the native Activity on Android, a Wails
// window elsewhere.
export async function loginYouTubeWithBrowser(): Promise<void> {
	try {
		const native = nativeBridge();
		if (native) await androidLogin(native);
		else await YouTubeBrowserLogin();
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
