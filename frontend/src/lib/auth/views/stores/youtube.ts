import {
	ClearYouTubeCookies,
	CookiesBrowsers,
	SetYouTubeCookies,
	SetYouTubeSession,
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
import { queryClient } from "@/lib/shared/views/ui/query-client";

export const youtubeSignedIn = signal(false);

// Who the stored session belongs to, shown next to "Connected as" the way the
// Spotify panel names its user. Empty when YouTube cannot say (a partial
// session), and then the panel only says it is signed in.
export const youtubeAccount = signal("");

// A session started from the app window carries the signing cookies but not the
// private ones, so the account's own lists stay closed. The panel says so.
export const youtubeSessionFull = signal(true);

// Desktop opens a window on youtube.com. Android is a single fullscreen WebView
// that cannot load an external URL, so it opens a separate activity instead;
// both hand the account's cookies back, so the panel offers the same button.
export const youtubeBrowserLoginSupported = signal(false);

const ANDROID = /Android/i.test(navigator.userAgent);

type Session = { cookies: string; visitor: string; version: string };
type LoginBridge = { loginYouTube(id: string): void };

declare global {
	interface Window {
		// Answers the Android sign-in started by loginYouTube.
		__jotaYouTubeCookies?: (
			id: string,
			cookies: string | null,
			visitor: string | null,
			version: string | null,
		) => void;
	}
}

const waiting = new Map<string, (session: Session) => void>();
let calls = 0;

window.__jotaYouTubeCookies = (id, cookies, visitor, version) => {
	const settle = waiting.get(id);
	if (!settle) return;
	waiting.delete(id);
	settle({
		cookies: cookies ?? "",
		visitor: visitor ?? "",
		version: version ?? "",
	});
};

function loginBridge(): LoginBridge | null {
	if (!ANDROID) return null;
	return (window as unknown as { wails?: LoginBridge }).wails ?? null;
}

// The sign-in activity runs in its own WebView, so the page gets the answer
// through a callback: a synchronous call would freeze this one until the user
// finished signing in.
function cookiesFromWindow(bridge: LoginBridge): Promise<Session> {
	return new Promise((resolve, reject) => {
		const id = String(++calls);
		const timer = setTimeout(
			() => {
				waiting.delete(id);
				reject(new Error(t("auth.youtube.loginTimeout")));
			},
			5 * 60 * 1000,
		);
		waiting.set(id, (session) => {
			clearTimeout(timer);
			if (session.cookies) resolve(session);
			else reject(new Error(t("auth.youtube.loginClosed")));
		});
		bridge.loginYouTube(id);
	});
}

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
	youtubeBrowserLoginSupported.value = supported || loginBridge() !== null;
	youtubeAccount.value = account ?? "";
	if (youtubeSignedIn.value) youtubeSignInSuggested.value = false;
}

// Storing account cookies is what unlocks age-restricted videos and avoids the
// bot check for that account's session. The UI must show the ban risk first.
export async function saveYouTubeCookies(cookies: string): Promise<void> {
	await SetYouTubeCookies(cookies);
	await syncYouTubeStatus();
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
		const bridge = loginBridge();
		if (bridge) {
			const session = await cookiesFromWindow(bridge);
			await SetYouTubeSession(
				session.cookies,
				session.visitor,
				session.version,
			);
		} else {
			await YouTubeBrowserLogin();
		}
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
