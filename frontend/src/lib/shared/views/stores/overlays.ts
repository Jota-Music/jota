import { signal } from "@preact/signals";

// Overlays (sheets, modals) live in signal state, not in the URL, so a
// history pop would navigate the route underneath an open one. Every open
// overlay holds its close here, and a back press dismisses the topmost
// instead. The Android hardware button reaches this through
// `window.__jotaBack`; the header back button calls `dismiss` directly.
const closers: Array<() => void> = [];

// Mirrors the depth so views can react: the header back button is enabled
// wherever a back press would close something rather than leave the route.
export const held = signal(0);

export function hold(close: () => void): () => void {
	closers.push(close);
	held.value = closers.length;
	return () => {
		const i = closers.indexOf(close);
		if (i >= 0) closers.splice(i, 1);
		held.value = closers.length;
	};
}

// Pops before calling: the release only runs on the next render, and a
// back press repeated inside the sheet's exit animation must fall through to
// history instead of re-closing the same overlay.
export function dismiss(): boolean {
	const close = closers.pop();
	held.value = closers.length;
	if (!close) return false;
	close();
	return true;
}
