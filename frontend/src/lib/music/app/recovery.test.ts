import { expect, test } from "bun:test";
import { playing, retryDelay, stalled, tick } from "@/lib/music/app/recovery";

test("a brief buffering pause is not a stall", () => {
	expect(stalled(2000)).toBe(false);
});

test("a stream that stops advancing is a stall", () => {
	expect(stalled(4000)).toBe(true);
});

test("retry backoff doubles", () => {
	expect(retryDelay(0)).toBe(1000);
	expect(retryDelay(1)).toBe(2000);
	expect(retryDelay(2)).toBe(4000);
});

test("retry backoff is capped", () => {
	expect(retryDelay(10)).toBe(15000);
});

// A fake element needs only what a wait touches; the real HTMLMediaElement is
// structurally compatible with it.
function fake(currentTime: number) {
	const listeners = new Map<string, Set<() => void>>();
	return {
		currentTime,
		addEventListener(name: string, listener: () => void) {
			const set = listeners.get(name) ?? new Set<() => void>();
			set.add(listener);
			listeners.set(name, set);
		},
		removeEventListener(name: string, listener: () => void) {
			listeners.get(name)?.delete(listener);
		},
		fire(name: string) {
			for (const listener of [...(listeners.get(name) ?? [])]) listener();
		},
	};
}

test("a stream that plays real audio is playing", async () => {
	const el = fake(10);
	const started = playing(el, 300);
	el.currentTime = 11.2;
	tick();
	expect(await started).toBe(true);
});

test("the last buffered frames of a dead stream are not playing", async () => {
	// Measured on the phone: paused=false, buffer full, no error, the clock moved
	// a tenth of a second and stopped. Taking that for playback left the session
	// paused for good, which is the bug.
	const el = fake(22.75);
	const started = playing(el, 0);
	el.currentTime = 22.854;
	tick();
	expect(await started).toBe(false);
});

test("an element that errors never started", async () => {
	const el = fake(10);
	const started = playing(el, 200);
	el.fire("error");
	expect(await started).toBe(false);
});

test("a play that resolves and stays silent is not playing", async () => {
	const silent = playing(fake(3), 0);
	tick();
	expect(await silent).toBe(false);
});
