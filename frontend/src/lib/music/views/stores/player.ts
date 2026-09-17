import type { Song } from "@/lib/music/model";
import {
	audioDuration,
	clampSeconds,
	currentSong,
	dragSeeking,
	hasLoadedAudio,
	play,
	playbackBlocked,
	progress,
	seek,
	setOnTrackEnded,
	stopPlayer,
	togglePlayPause,
} from "@/lib/music/views/stores/audio";
import { AudioCache } from "@/lib/music/views/stores/cache";
import * as media from "@/lib/music/views/stores/media-session";
import {
	currentIndex,
	persistQueue,
	queue,
	repeat,
	shuffle,
} from "@/lib/music/views/stores/queue";
import { playGate, publish } from "@/lib/music/views/stores/remote";

// Resolve this many tracks ahead: an already-resolved stream keeps playing while
// the YouTube API is blocked, so a wider window rides out longer outages.
const PRELOAD_AHEAD = 4;

export function preloadUpcomingSongs(songs: Song[], idx: number) {
	const upcoming = songs.slice(idx + 1, idx + 1 + PRELOAD_AHEAD);
	if (upcoming.length > 0) void AudioCache.preload(...upcoming);
}

export function seekFromLocalControl(seconds: number) {
	const clamped = clampSeconds(seconds, audioDuration.value);
	publish({ action: "seek", positionMs: clamped * 1000 });
	seek(clamped);
}

export function previewSeek(seconds: number) {
	dragSeeking.value = true;
	progress.value = clampSeconds(seconds, audioDuration.value);
}

export function commitSeek(seconds: number) {
	dragSeeking.value = false;
	seekFromLocalControl(seconds);
}

function setQueueState(nextQueue: Song[], nextIndex: number) {
	const song =
		nextIndex >= 0 && nextIndex < nextQueue.length
			? nextQueue[nextIndex]
			: null;

	queue.value = nextQueue;
	currentIndex.value = nextIndex;
	persistQueue();

	if (!song) {
		stopPlayer();
		return;
	}

	currentSong.value = song;
	preloadUpcomingSongs(nextQueue, nextIndex);
}

// Pick the next queued song that has not failed yet in this sweep, winding
// around from `from`. Returns null once every song has been tried, so a totally
// unavailable queue stops instead of looping forever.
function nextUntried(
	length: number,
	from: number,
	tried: Set<number>,
): number | null {
	for (let step = 1; step < length; step++) {
		const idx = (from + step) % length;
		if (!tried.has(idx)) return idx;
	}
	return null;
}

async function playAtIndex(
	i: number,
	tried = new Set<number>(),
): Promise<void> {
	const q = queue.value;
	if (i < 0 || i >= q.length) return;
	const song = q[i];

	currentIndex.value = i;
	currentSong.value = song;
	persistQueue();

	// In a room every member runs the same load round: whoever pressed the track
	// (or auto-advanced) announces it and the relay releases a shared start, so
	// nobody begins ahead of the others.
	const gate = playGate.value;
	if (gate) {
		await gate(song);
		return;
	}

	// play() exhausts this track's recovery ladder before returning false.
	const ok = await play(song);
	if (queue.value[currentIndex.value]?.id !== song.id) return;

	if (!ok) {
		// WebKit blocked playback for lack of user activation: the track is fine,
		// so keep the queue here and let the user interact instead of sweeping
		// every song through the recovery ladder.
		if (playbackBlocked()) return;

		tried.add(i);
		const next = nextUntried(queue.value.length, i, tried);
		if (next != null) {
			void playAtIndex(next, tried);
		} else {
			stopPlayer();
		}
		return;
	}

	preloadUpcomingSongs(q, i);
}

export async function playFromQueueSelection(
	fullOrderedSongs: Song[],
	clicked: Song,
) {
	const i = fullOrderedSongs.findIndex((s) => s.id === clicked.id);
	if (i === -1) return;

	await setQueueState(fullOrderedSongs.slice(), i);
	await playAtIndex(i);
}

export function enqueue(song: Song) {
	const newQueue = [...queue.value];
	const newIndex = currentIndex.value;

	if (newQueue.length === 0) {
		newQueue.push(song);
		queue.value = newQueue;
		currentIndex.value = 0;
		persistQueue();
		void AudioCache.preload(song);
		return;
	}

	const insertAt =
		newIndex >= 0 && newIndex < newQueue.length
			? newIndex + 1
			: newQueue.length;

	newQueue.splice(insertAt, 0, song);
	queue.value = newQueue;
	persistQueue();

	void AudioCache.preload(song);
}

export async function unqueue(removeIdx: number): Promise<void> {
	const q = [...queue.value];
	const n = q.length;
	if (removeIdx < 0 || removeIdx >= n) return;

	const playingId =
		currentIndex.value >= 0 && currentIndex.value < n
			? q[currentIndex.value].id
			: null;

	q.splice(removeIdx, 1);

	let nextIndex: number;

	if (q.length === 0) {
		nextIndex = -1;
	} else if (playingId != null) {
		const j = q.findIndex((s) => s.id === playingId);
		nextIndex = j !== -1 ? j : Math.min(removeIdx, q.length - 1);
	} else {
		nextIndex = 0;
	}

	setQueueState(q, nextIndex);
}

export async function moveQueue(from: number, to: number): Promise<void> {
	const a = [...queue.value];
	const n = a.length;
	if (from < 0 || from >= n) return;

	const clamped = Math.max(0, Math.min(n - 1, to));
	if (from === clamped) return;

	const playingId =
		currentIndex.value >= 0 && currentIndex.value < n
			? a[currentIndex.value].id
			: null;

	const [it] = a.splice(from, 1);
	a.splice(clamped, 0, it);

	const nextIndex =
		playingId != null
			? a.findIndex((s) => s.id === playingId)
			: a.length > 0
				? 0
				: -1;

	setQueueState(a, nextIndex >= 0 ? nextIndex : a.length > 0 ? 0 : -1);
}

export async function moveAfterCurrent(from: number): Promise<void> {
	const a = [...queue.value];
	const n = a.length;
	const ci = currentIndex.value;

	if (from < 0 || from >= n) return;
	if (ci < 0 || ci >= n) return;
	if (from === ci || from === ci + 1) return;

	const playingId = a[ci].id;

	const [it] = a.splice(from, 1);

	const ci2 = a.findIndex((s) => s.id === playingId);
	if (ci2 === -1) return;

	const insertAt = Math.min(ci2 + 1, a.length);
	a.splice(insertAt, 0, it);

	const nextIndex = a.findIndex((s) => s.id === playingId);

	setQueueState(a, nextIndex >= 0 ? nextIndex : 0);
}

export async function playAt(i: number): Promise<void> {
	const q = queue.value;

	if (i < 0 || i >= q.length) return;
	if (
		i === currentIndex.value &&
		currentSong.value?.id === q[i].id &&
		hasLoadedAudio()
	) {
		return;
	}

	await playAtIndex(i);
}

export async function toggleSong() {
	publish({ action: "toggle" });
	await togglePlayPause();
}

function pickRandom(songs: Song[], i: number): number | null {
	const candidates = songs.filter((_, idx) => idx !== i);
	if (candidates.length === 0) return null;
	const pick = candidates[Math.floor(Math.random() * candidates.length)];
	return songs.findIndex((song) => song.id === pick.id);
}

function pick(
	songs: Song[],
	i: number,
	mode: "off" | "all" | "one",
	shuffleOn: boolean,
	step: 1 | -1,
): number | null {
	if (mode === "one") return i;
	if (shuffleOn) return pickRandom(songs, i);
	const next = i + step;
	if (next < 0 || next > songs.length - 1) {
		return mode === "all" ? (step > 0 ? 0 : songs.length - 1) : null;
	}
	return next;
}

export async function nextSong() {
	const q = queue.value;
	if (q.length === 0) return;
	const next = pick(q, currentIndex.value, repeat.value, shuffle.value, 1);
	if (next == null) return;
	await playAtIndex(next);
}

export async function prevSong() {
	const q = queue.value;
	if (q.length === 0) return;
	const prev = pick(q, currentIndex.value, repeat.value, shuffle.value, -1);
	if (prev == null) return;
	await playAtIndex(prev);
}

setOnTrackEnded(() => {
	const q = queue.value;
	if (q.length === 0) return;
	const next = pick(q, currentIndex.value, repeat.value, shuffle.value, 1);
	if (next == null) return;
	void playAtIndex(next);
});

media.tracks({
	next: () => void nextSong(),
	previous: () => void prevSong(),
});
