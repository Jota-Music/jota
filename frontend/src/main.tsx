import { effect } from "@preact/signals";
import { render } from "preact";
import {
	spotifyUser,
	syncSpotifyStatus,
} from "@/lib/auth/views/stores/session";
import {
	syncYouTubeStatus,
	watchIncoming,
	youtubeSignedIn,
} from "@/lib/auth/views/stores/youtube";
import { startAutosave } from "@/lib/music/app/backup";
import { warm, warmOrder, warmSources } from "@/lib/music/app/prefetch";
import { t } from "@/lib/shared/i18n";
import { logError } from "@/lib/shared/views/stores/errors";
import { dismiss } from "@/lib/shared/views/stores/overlays";
import { Boundary } from "@/lib/shared/views/ui/components/boundary";
import Router from "@/lib/shared/views/ui/router";
import "@wailsio/runtime";

// Preact 10.29 has no options.devtools: preact/debug installs the warnings and
// pulls in preact/devtools, which attaches to the extension when one is present.
if (import.meta.env.DEV) {
	await import("preact/debug");
}

const description = document.querySelector('meta[name="description"]');
effect(() => {
	document.title = t("app.title");
	description?.setAttribute("content", t("app.description"));
});

// console.error is where unexpected failures land; mirror it to the log file.
// console.warn/log are left alone: they are mostly benign in this app.
const nativeConsoleError = console.error.bind(console);
console.error = (...args: unknown[]) => {
	nativeConsoleError(...args);
	logError(args.length === 1 ? args[0] : args.join(" "));
};

window.addEventListener("error", (e) =>
	logError(e.error ?? e.message, "uncaught"),
);
window.addEventListener("unhandledrejection", (e) =>
	logError(e.reason, "unhandled rejection"),
);

declare global {
	interface Window {
		__jotaBack?: () => boolean;
	}
}

// The Android back button would otherwise walk WebView history and swap the
// route out from under an open sheet; it asks the page first and only goes
// back when nothing was holding.
window.__jotaBack = dismiss;

// Coming back to the app, and on a timer while it is open, is when the sync
// folder is reconciled; the settings switch decides whether that happens.
startAutosave();

// A session export opened with Jota (Android "open with") lands here.
watchIncoming();

const appRoot = document.getElementById("app");
if (!appRoot) throw new Error("Missing #app root");

// How long the first render may wait on a source's own list. Long enough for a
// cache to answer on this machine, short enough that a source which has to fetch
// never holds the window.
const sourceWait = 120;

function deadline(ms: number): Promise<void> {
	return new Promise((resolve) => {
		setTimeout(resolve, ms);
	});
}

// The library's own records are read before the first render, so the shelf
// paints them together: asking after the paint filled the grid in one source at
// a time. The reads are the store in memory, and the boot mark in index.html
// covers the wait when it outlives the mount.
await Promise.all([warm(), syncSpotifyStatus(), syncYouTubeStatus()]);

// The saved order is keyed by the account those readings name, and a shelf that
// reorders once it lands is a shelf that jumps. The sources' own lists are asked
// next: one first paint without them shows the library growing a tile at a time,
// so they get a short wait and no more. That wait is a cache answering; a source
// that has to fetch is a request, and a launch never waits on the network, so
// past the wait the shelf paints what this device holds.
await warmOrder(spotifyUser.value);
await Promise.race([
	warmSources(spotifyUser.value, youtubeSignedIn.value),
	deadline(sourceWait),
]);

render(
	<Boundary>
		<Router />
	</Boundary>,
	appRoot,
);
