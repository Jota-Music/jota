import { signal } from "@preact/signals";

// Mirrors the browser's position in its session history so the header can know
// whether going forward is possible (the History API offers no such probe).
export const canForward = signal(false);

const pushState = history.pushState.bind(history);
const replaceState = history.replaceState.bind(history);

// Every entry the app writes carries its own position. A native back/forward
// (the keyboard, a mouse button, a trackpad gesture) moves the browser without
// going through the wrappers below, and the entry is what tells the pointer
// where it landed instead of guessing the direction.
const indexKey = "jotaHistoryIndex";

let pointer = Math.max(history.length - 1, 0);
let size = Math.max(history.length, 1);

function sync() {
	canForward.value = pointer < size - 1;
}

function sameUrl(url: string | URL | null | undefined): boolean {
	if (url == null) return true;
	const target = new URL(url, window.location.href);
	const current = new URL(window.location.href);
	return (
		target.pathname + target.search + target.hash ===
		current.pathname + current.search + current.hash
	);
}

function withIndex(state: unknown, at: number): Record<string, unknown> {
	const base = typeof state === "object" && state !== null ? state : {};
	return { ...base, [indexKey]: at };
}

function indexOf(state: unknown): number | null {
	if (typeof state !== "object" || state === null) return null;
	const at = (state as Record<string, unknown>)[indexKey];
	return typeof at === "number" ? at : null;
}

history.pushState = (state, title, url) => {
	const changed = !sameUrl(url);
	const at = changed ? pointer + 1 : pointer;
	pushState(withIndex(state, at), title, url);
	if (!changed) return;
	pointer = at;
	size = pointer + 1;
	sync();
};

history.replaceState = (state, title, url) => {
	replaceState(withIndex(state, pointer), title, url);
};

window.addEventListener("popstate", (event) => {
	const at = indexOf(event.state);
	if (at === null) return;
	pointer = at;
	size = Math.max(size, pointer + 1);
	sync();
});

// The entry the app opened on carries no index of ours, so a back that lands on
// it would leave the pointer where it was.
replaceState(withIndex(history.state, pointer), "", window.location.href);

export function back(): boolean {
	if (pointer <= 0) return false;
	pointer -= 1;
	sync();
	history.back();
	return true;
}

export function forward(): boolean {
	if (pointer >= size - 1) return false;
	pointer += 1;
	sync();
	history.forward();
	return true;
}
