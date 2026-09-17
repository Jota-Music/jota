import type { RepeatMode, Song } from "@/lib/music/model";

// The queue is the play order (a stable shuffle when shuffle is on), so moving
// through it is always linear.
export function pick(
	songs: Song[],
	i: number,
	mode: RepeatMode,
	step: 1 | -1,
): number | null {
	if (mode === "one") return i;
	if (songs.length === 0) return null;
	const next = i + step;
	if (next < 0 || next > songs.length - 1) {
		return mode === "all" ? (step > 0 ? 0 : songs.length - 1) : null;
	}
	return next;
}
