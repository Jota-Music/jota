import { signal } from "@preact/signals";
import { Clipboard } from "@wailsio/runtime";
import { type ClipboardLink, clipboardLink } from "@/lib/clipboard/app/link";

export type Pending = {
	link: ClipboardLink;
	dismissed: boolean;
};

export const pending = signal<Pending | null>(null);

const ignored = new Set<string>();
const POLL_MS = 1500;
const MOBILE_POLL_MS = 5000;

let watching = false;
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
	try {
		return await Clipboard.Text();
	} catch {
		try {
			return await navigator.clipboard.readText();
		} catch {
			return null;
		}
	}
}

async function check(): Promise<void> {
	const text = await read();
	if (!text || text === last) return;
	last = text;

	const link = clipboardLink(text);
	if (!link || ignored.has(link.text)) return;

	ignored.add(link.text);
	pending.value = { link, dismissed: false };
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
	const mobile =
		typeof window !== "undefined" &&
		window.matchMedia("(pointer: coarse)").matches;
	const interval = mobile ? MOBILE_POLL_MS : POLL_MS;

	window.setInterval(() => {
		if (document.visibilityState === "visible") void check();
	}, interval);
}
