import { signal } from "@preact/signals";

// Mirrors the browser's position in its session history so the header can know
// whether going forward is possible (the History API offers no such probe).
export const canForward = signal(false);

const pushState = history.pushState.bind(history);
const replaceState = history.replaceState.bind(history);

let pointer = Math.max(history.length - 1, 0);
let size = Math.max(history.length, 1);

function sync() {
	canForward.value = pointer < size - 1;
}

function sameUrl(url: string | URL | null): boolean {
	if (url == null) return false;
	const target = new URL(url, window.location.href);
	const current = new URL(window.location.href);
	return target.pathname + target.search + target.hash ===
		current.pathname + current.search + current.hash;
}

history.pushState = (state, title, url) => {
	if (sameUrl(url)) return;
	pushState(state, title, url);
	pointer += 1;
	size = pointer + 1;
	sync();
};

history.replaceState = (state, title, url) => {
	if (sameUrl(url)) return;
	replaceState(state, title, url);
};

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