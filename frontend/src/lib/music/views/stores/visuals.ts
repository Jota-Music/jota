import { signal } from "@preact/signals";
import * as storage from "@/lib/shared/utils/storage";

const KEY = "visuals-backdrop";

// On unless it was turned off: the glow is what the app looks like, so a
// fresh install should not have to find a switch to get it.
export const backdrop = signal(storage.get<string>(KEY, "1") !== "0");

export function setBackdrop(on: boolean): void {
	backdrop.value = on;
	storage.set(KEY, on ? "1" : "0");
}
