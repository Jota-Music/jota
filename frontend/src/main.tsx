import { effect } from "@preact/signals";
import { render } from "preact";
import { watchIncoming } from "@/lib/auth/views/stores/youtube";
import { startAutosave } from "@/lib/music/app/backup";
import { t } from "@/lib/shared/i18n";
import { logError } from "@/lib/shared/views/stores/errors";
import { dismiss } from "@/lib/shared/views/stores/overlays";
import { Boundary } from "@/lib/shared/views/ui/components/boundary";
import Router from "@/lib/shared/views/ui/router";
import "@wailsio/runtime";
import "@/style.tw.css";

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
render(
	<Boundary>
		<Router />
	</Boundary>,
	appRoot,
);
