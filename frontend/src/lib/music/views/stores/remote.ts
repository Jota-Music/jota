import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import type { ControlAction } from "@/lib/sync/model";

export const remoteControl = signal<((a: ControlAction) => void) | null>(null);

export const playGate = signal<((song: Song) => Promise<void>) | null>(null);

export function forward(a: ControlAction): void {
	remoteControl.value?.(a);
}
