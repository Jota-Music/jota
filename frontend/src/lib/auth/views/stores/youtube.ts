import {
	ClearYouTubeCookies,
	CookiesBrowsers,
	ExportYouTubeCookiesFile,
	ImportYouTubeCookies,
	SessionPass,
	SetYouTubeCookies,
	YouTubeAccount,
	YouTubeBrowserAccounts,
	YouTubeBrowserLogin,
	YouTubeBrowserLoginSupported,
	YouTubeCookiesFromBrowser,
	YouTubeSessionFull,
	YouTubeSignedIn,
} from "@bindings/app";
import { signal } from "@preact/signals";
import { Events } from "@wailsio/runtime";

import { t } from "@/lib/shared/i18n";
import { addError, notify } from "@/lib/shared/views/stores/errors";
import { queryClient } from "@/lib/shared/views/ui/query-client";

export const youtubeSignedIn = signal(false);

// Who the stored session belongs to, shown next to "Connected as" the way the
// Spotify panel names its user. Empty when YouTube cannot say (a partial
// session), and then the panel only says it is signed in.
export const youtubeAccount = signal("");

// A session started from the app window carries the signing cookies but not the
// private ones, so the account's own lists stay closed. The panel says so.
export const youtubeSessionFull = signal(true);

// Desktop opens a window on youtube.com and hands the account's cookies back,
// so the panel offers the same button. Android has no in-app sign-in; it relies
// on pasting or importing a cookies.txt instead.
export const youtubeBrowserLoginSupported = signal(false);

// Set when a track fails because it needs a signed-in YouTube session.
export const youtubeSignInSuggested = signal(false);

Events.On("youtube:signin-required", () => {
	if (!youtubeSignedIn.value) youtubeSignInSuggested.value = true;
});

export function dismissYouTubeSignInSuggestion(): void {
	youtubeSignInSuggested.value = false;
}

export async function syncYouTubeStatus(): Promise<void> {
	// Independent reads, so both crosses the bridge at once instead of
	// stacking two round trips on every mount.
	const [signedIn, full, supported, account] = await Promise.all([
		YouTubeSignedIn().catch(() => false),
		YouTubeSessionFull().catch(() => true),
		YouTubeBrowserLoginSupported().catch(() => false),
		YouTubeAccount().catch(() => ""),
	]);
	youtubeSignedIn.value = signedIn;
	youtubeSessionFull.value = full;
	youtubeBrowserLoginSupported.value = supported;
	youtubeAccount.value = account ?? "";
	if (youtubeSignedIn.value) youtubeSignInSuggested.value = false;
}

// Storing account cookies is what unlocks age-restricted videos and avoids the
// bot check for that account's session. The UI must show the ban risk first.
export async function saveYouTubeCookies(cookies: string): Promise<void> {
	await SetYouTubeCookies(cookies);
	await syncYouTubeStatus();
}

export async function exportYouTubeCookiesFile(): Promise<string> {
	return await ExportYouTubeCookiesFile();
}

export async function importYouTubeCookies(
	data: string,
	pass: string,
): Promise<void> {
	await ImportYouTubeCookies(data, pass);
	await syncYouTubeStatus();
}

// A file another app opened Jota with (Android "open with" on the export this
// device sent itself). The native side keeps the text until this takes it, so a
// cold start finds it on mount and a running app is woken by __jotaIncoming.
type IncomingBridge = {
	takeIncomingFile?: () => string;
	takeIncomingError?: () => string;
};

declare global {
	interface Window {
		__jotaIncoming?: () => void;
	}
}

async function importIncoming(): Promise<void> {
	const bridge = (window as unknown as { wails?: IncomingBridge }).wails;
	if (bridge?.takeIncomingError?.()) {
		addError(new Error(t("auth.youtube.unreadable")));
		return;
	}
	const text = bridge?.takeIncomingFile?.();
	if (!text) return;
	try {
		await importYouTubeCookies(text, (await SessionPass()) ?? "");
		notify(t("auth.youtube.imported"));
	} catch (error) {
		addError(error);
	}
}

export function watchIncoming(): void {
	if (typeof window === "undefined") return;
	window.__jotaIncoming = () => void importIncoming();
	void importIncoming();
	// The native wake can land before the page finished loading on a cold start;
	// coming back to the app is the second chance to pick the file up.
	document.addEventListener("visibilitychange", () => {
		if (!document.hidden) void importIncoming();
	});
}

// The browsers on this machine whose cookie jar can be read, so the panel only
// offers an import it can actually perform.
export async function browsers(): Promise<string[]> {
	try {
		return (await CookiesBrowsers()) ?? [];
	} catch {
		return [];
	}
}

// The account each of those jars holds. Two browsers can both have a session and
// still be different Google accounts, so the panel labels them with it.
export async function browserAccounts(): Promise<Record<string, string>> {
	try {
		const found = (await YouTubeBrowserAccounts()) ?? {};
		const labelless: Record<string, string> = {};
		for (const [name, account] of Object.entries(found)) {
			if (account) labelless[name] = account;
		}
		return labelless;
	} catch {
		return {};
	}
}

// Signs in with the session the browser already holds, so nothing is pasted and
// no window opens. The jar is read and verified on the backend.
export async function signInFromBrowser(browser: string): Promise<void> {
	try {
		await YouTubeCookiesFromBrowser(browser);
	} finally {
		await syncYouTubeStatus();
	}
}

// Opens a window on youtube.com so the user can sign in normally; the page
// hands its cookies back to the app. Android opens its own activity and hands
// the cookies, the visitor and the client version through the bridge, and both
// paths store and verify them here.
export async function loginYouTubeWithBrowser(): Promise<void> {
	try {
		await YouTubeBrowserLogin();
	} finally {
		await syncYouTubeStatus();
	}
	await queryClient.invalidateQueries({
		queryKey: ["user-playlists", "youtube"],
	});
}

export async function signOutYouTube(): Promise<void> {
	try {
		await ClearYouTubeCookies();
	} catch {}
	await syncYouTubeStatus();
}
