import type { QueueSong, RepeatMode, Song } from "@/lib/music/model";

function pickRandom(songs: Song[], i: number): number | null {
	const candidates = songs.filter((_, idx) => idx !== i);
	if (candidates.length === 0) return null;
	const pick = candidates[Math.floor(Math.random() * candidates.length)];
	return songs.findIndex((song) => song.id === pick.id);
}

// First explicitly queued song after `i`, wrapping to the front so shuffle
// never skips it.
export function nextQueued(songs: QueueSong[], i: number): number | null {
	for (let step = 1; step < songs.length; step++) {
		const idx = (i + step) % songs.length;
		if (songs[idx].queued) return idx;
	}
	return null;
}

export function pick(
	songs: QueueSong[],
	i: number,
	mode: RepeatMode,
	shuffleOn: boolean,
	step: 1 | -1,
): number | null {
	if (mode === "one") return i;
	if (shuffleOn) {
		if (step > 0) {
			const queued = nextQueued(songs, i);
			if (queued != null) return queued;
		}
		return pickRandom(songs, i);
	}
	const next = i + step;
	if (next < 0 || next > songs.length - 1) {
		return mode === "all" ? (step > 0 ? 0 : songs.length - 1) : null;
	}
	return next;
}
