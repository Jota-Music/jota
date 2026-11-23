import { effect } from "@preact/signals";
import { System, Window as WailsWindow } from "@wailsio/runtime";
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
import { Tooltip } from "@/lib/shared/views/ui/components/tooltip";
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
		// When the page's markup was up, set by the inline script in index.html.
		__jotaBoot?: number;
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

// The boot mark in index.html copies the app's bar, and that bar's window
// buttons belong to a component that is not mounted yet: the markup is static,
// so the click is answered here. It reads the attribute, so it does nothing once
// the real header is on screen and the boot node is gone.
if (System.IsDesktop()) {
	document.addEventListener("click", (event) => {
		const target = event.target;
		if (!(target instanceof Element)) return;
		switch (target.closest("[data-window]")?.getAttribute("data-window")) {
			case "minimise":
				void WailsWindow.Minimise();
				break;
			case "maximise":
				void WailsWindow.Maximise();
				break;
			case "close":
				void WailsWindow.Close();
				break;
		}
	});
}

const appRoot = document.getElementById("app");
if (!appRoot) throw new Error("Missing #app root");

// How long the first render may wait on the store and the sources' cached lists.
// A read of this device answers well inside it; anything slower is a fetch, and a
// launch never waits on the network. Past the wait the shelf paints what this
// device holds and the answer lands late, so the window is never left with
// nothing to click.
const bootWait = 120;

function deadline(ms: number): Promise<void> {
	return new Promise((resolve) => {
		setTimeout(resolve, ms);
	});
}

// One line per launch, the same shape as the backend's "window ready in": the
// phase each mark names, read from the page's own clock. A launch that gets slow
// again then points at the wait instead of only being felt.
const marks = [`page=${Math.round(window.__jotaBoot ?? 0)}`];
function mark(name: string): void {
	marks.push(`${name}=${Math.round(performance.now())}`);
}
mark("module");

// The YouTube status is where a launch used to reach the network: the account's
// own name is a request the first time it is asked, and a launch never waits on
// one. It gets no wait at all, so the shelf paints from this device and the
// panel fills in when the answer lands.
const youtubeStatus = syncYouTubeStatus();

// The library's own records are read before the first render, so the shelf
// paints them together: asking after the paint filled the grid in one source at
// a time. The saved order is keyed by the account those readings name, and the
// sources' own lists are asked last, so the shelf never reorders under the
// pointer. The reads are the store in memory, but the whole wait is bounded: a
// read that never answers is a window that never becomes interactive.
let warmed = false;
await Promise.race([
	(async () => {
		await Promise.all([warm(), syncSpotifyStatus()]);
		mark("library");
		await warmOrder(spotifyUser.value);
		await warmSources(spotifyUser.value, youtubeSignedIn.value);
		mark("sources");
		warmed = true;
	})(),
	deadline(bootWait),
]);
if (!warmed) marks.push("warm=miss");

// The session the first ask carried was the one this device had before the
// answer: a status that lands late is worth asking the sources' lists again for.
void youtubeStatus.then(() =>
	warmSources(spotifyUser.value, youtubeSignedIn.value),
);

render(
	<Boundary>
		<Router />
		<Tooltip />
	</Boundary>,
	appRoot,
);
mark("render");

// The first frame is where the app is on screen and the tick after it is where
// the main thread is free to answer input: the line separates a slow read from a
// slow first paint.
requestAnimationFrame(() => {
	mark("frame");
	setTimeout(() => {
		mark("idle");
		logError(marks.join(" "), "boot");
	}, 0);
});
