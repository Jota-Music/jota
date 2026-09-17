import { effect } from "@preact/signals";
import { render } from "preact";
import { t } from "@/lib/shared/i18n";
import { logError } from "@/lib/shared/views/stores/errors";
import Router from "@/lib/shared/views/ui/router";
import "@wailsio/runtime";
import "./style.tw.css";

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

const appRoot = document.getElementById("app");
if (!appRoot) throw new Error("Missing #app root");
render(<Router />, appRoot);
