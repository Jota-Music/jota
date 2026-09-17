import { signal } from "@preact/signals";
import type { ControlAction, Song } from "@/lib/music/model";

// The room surface. `control` broadcasts a local control; `playGate` runs the
// load round for a track change. Both are null outside a room, where the caller
// just applies locally.
export const control = signal<((a: ControlAction) => void) | null>(null);

export const playGate = signal<((song: Song) => Promise<void>) | null>(null);

export function publish(a: ControlAction): void {
	control.value?.(a);
}
