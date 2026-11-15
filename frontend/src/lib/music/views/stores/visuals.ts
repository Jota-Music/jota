import { computed, signal } from "@preact/signals";
import * as storage from "@/lib/shared/utils/storage";

const BACKDROP_KEY = "visuals-backdrop";
const STRENGTH_KEY = "visuals-strength";
const MOTE_KEY = "visuals-mote-size";

export const defaults = { backdrop: true, strength: 0.12, moteSize: 1 };

// The field's percentage is the user's unit, not the dust's radius: 100% is
// the size the glow reads best at, 1.25x the radius the field paints.
const MOTE_SCALE = 1.25;

// On unless it was turned off: the glow is what the app looks like, so a
// fresh install should not have to find a switch to get it.
export const backdrop = signal(
	storage.get<string>(BACKDROP_KEY, defaults.backdrop ? "1" : "0") !== "0",
);

// A knob is the user's number, not ours: how strong the glow reads and how big
// the dust is both depend on the screen it is seen on.
function knob(key: string, fallback: number) {
	const saved = Number(storage.get<string>(key, ""));
	return signal(Number.isFinite(saved) && saved > 0 ? saved : fallback);
}

export const strength = knob(STRENGTH_KEY, defaults.strength);
export const moteSize = knob(MOTE_KEY, defaults.moteSize);

export const moteScale = computed(() => moteSize.value * MOTE_SCALE);

export function setBackdrop(on: boolean): void {
	backdrop.value = on;
	storage.set(BACKDROP_KEY, on ? "1" : "0");
}

export function setStrength(value: number): void {
	strength.value = value;
	storage.set(STRENGTH_KEY, String(value));
}

export function setMoteSize(value: number): void {
	moteSize.value = value;
	storage.set(MOTE_KEY, String(value));
}
