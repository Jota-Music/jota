import { ClipboardText } from "@bindings/app";
import { signal } from "@preact/signals";
import { Clipboard } from "@wailsio/runtime";
import { type ClipboardLink, clipboardLink } from "@/lib/clipboard/app/link";
import { within } from "@/lib/clipboard/app/within";

export type Pending = {
	link: ClipboardLink;
	dismissed: boolean;
};

export const pending = signal<Pending | null>(null);

const ignored = new Set<string>();
const MOBILE_POLL_MS = 5000;
// Above the backend's own 400ms bound, so a slow desktop owner is not mistaken
// for a wedged transport.
const READ_TIMEOUT_MS = 600;

let watching = false;
let busy = false;
let last = "";

type NativeBridge = { getClipboardText?: () => string };

function native(): NativeBridge | null {
	if (typeof window === "undefined") return null;
	const wails = (window as unknown as { wails?: NativeBridge }).wails;
	return wails && typeof wails.getClipboardText === "function" ? wails : null;
}

async function read(): Promise<string | null> {
	const bridge = native();
	if (bridge?.getClipboardText) {
		try {
			const text = bridge.getClipboardText();
			if (text) return text;
		} catch {
			// fall through to other providers
		}
	}
	// GTK's blocking read freezes the window when the selection owner does not
	// answer (an XWayland client waiting on the compositor's bridge), so the
	// backend asks asynchronously and answers empty instead. Where that read is
	// unavailable the runtime's own call is still the only one, and it is not on
	// a GTK loop there.
	//
	// Every provider is bounded: a stalled transport must leave the poll free to
	// run again, not hold it until the window closes.
	const backend = await within(ClipboardText(), READ_TIMEOUT_MS);
	if (backend !== null) return backend;
	const runtime = await within(Clipboard.Text(), READ_TIMEOUT_MS);
	if (runtime !== null) return runtime;
	return await within(navigator.clipboard.readText(), READ_TIMEOUT_MS);
}

async function check(): Promise<void> {
	if (busy) return;
	busy = true;
	try {
		const text = await read();
		if (!text || text === last) return;
		last = text;

		const link = clipboardLink(text);
		if (!link || ignored.has(link.text)) return;

		ignored.add(link.text);
		pending.value = { link, dismissed: false };
	} finally {
		busy = false;
	}
}

export function remember(link: string): void {
	ignored.add(link.trim());
}

export function dismiss(): void {
	const current = pending.value;
	if (!current) return;
	ignored.add(current.link.text);
	pending.value = { ...current, dismissed: true };
}

export function watch(): void {
	if (watching) return;
	watching = true;

	const poll = () => {
		if (document.visibilityState === "visible") void check();
	};

	void check();
	window.addEventListener("focus", poll);
	document.addEventListener("visibilitychange", poll);
	// Android answers through the native bridge, which never touches GTK. The
	// desktop call lands in gtk_clipboard_wait_for_text, which blocks the GTK
	// thread until the selection owner answers: a timer there froze the window
	// when the poll met the shutdown. Focus is when a paste is plausible anyway.
	const mobile =
		typeof window !== "undefined" &&
		window.matchMedia("(pointer: coarse)").matches;
	if (mobile) window.setInterval(poll, MOBILE_POLL_MS);
}
