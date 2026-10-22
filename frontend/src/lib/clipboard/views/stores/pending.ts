import { signal } from "@preact/signals";
import { Clipboard, System } from "@wailsio/runtime";
import { type ClipboardLink, clipboardLink } from "@/lib/clipboard/app/link";

export type Pending = {
	link: ClipboardLink;
	dismissed: boolean;
};

export const pending = signal<Pending | null>(null);

const ignored = new Set<string>();
const POLL_MS = 1500;

let watching = false;
let last = "";

async function read(): Promise<string | null> {
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
	window.setInterval(() => {
		if (document.visibilityState === "visible" && System.IsDesktop()) {
			void check();
		}
	}, POLL_MS);
}
