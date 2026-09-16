import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import {
	audioDuration,
	currentSong,
	dragSeeking,
	hasLoadedAudio,
	play,
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
import { forward, playGate } from "@/lib/music/views/stores/remote";

export const autoAdvance = signal(true);

function clampPlaybackSeconds(seconds: number): number {
	if (!Number.isFinite(seconds) || seconds < 0) return 0;
	const d = audioDuration.value;
	if (Number.isFinite(d) && d > 0) {
		const eps = 0.05;
		return Math.min(seconds, Math.max(0, d - eps));
	}
	return seconds;
}

export function preloadUpcomingSongs(songs: Song[], idx: number) {
	const upcoming = songs.slice(idx + 1, idx + 3);
	if (upcoming.length > 0) void AudioCache.preload(...upcoming);
}

export function seekFromLocalControl(seconds: number) {
	const clamped = clampPlaybackSeconds(seconds);
	forward({ action: "seek", positionMs: clamped * 1000 });
	seek(clamped);
}

export function previewSeek(seconds: number) {
	dragSeeking.value = true;
	progress.value = clampPlaybackSeconds(seconds);
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

async function playAtIndex(i: number): Promise<void> {
	const q = queue.value;
	if (i < 0 || i >= q.length) return;
	const song = q[i];

	const gate = playGate.value;
	if (gate) await gate(song);
	if (queue.value[i]?.id !== song.id) return;

	currentIndex.value = i;
	currentSong.value = song;
	persistQueue();

	const ok = await play(song);
	if (queue.value[currentIndex.value]?.id !== song.id) return;
	if (!ok) {
		const next = pickNext(
			queue.value,
			currentIndex.value,
			repeat.value,
			shuffle.value,
		);
		if (next != null && next !== currentIndex.value) {
			void playAtIndex(next);
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

	forward({
		action: "playSelection",
		songId: clicked.id,
		songs: fullOrderedSongs,
	});

	await setQueueState(fullOrderedSongs.slice(), i);
	await playAtIndex(i);
}

export function enqueue(song: Song) {
	forward({ action: "enqueue", song });
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

	forward({ action: "remove", index: removeIdx });

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

	forward({ action: "move", from, to: clamped });

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

	forward({ action: "moveAfter", index: from });

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

	forward({ action: "play", index: i });

	await playAtIndex(i);
}

export async function toggleSong() {
	forward({ action: "toggle" });
	await togglePlayPause();
}

function pickRandom(songs: Song[], i: number): number | null {
	const candidates = songs.filter((_, idx) => idx !== i);
	if (candidates.length === 0) return null;
	const pick = candidates[Math.floor(Math.random() * candidates.length)];
	return songs.findIndex((song) => song.id === pick.id);
}

function pickNext(
	songs: Song[],
	i: number,
	mode: "off" | "all" | "one",
	shuffleOn: boolean,
): number | null {
	if (mode === "one") return i;
	if (shuffleOn) return pickRandom(songs, i);
	if (i >= songs.length - 1) return mode === "all" ? 0 : null;
	return i + 1;
}

function pickPrev(
	songs: Song[],
	i: number,
	mode: "off" | "all" | "one",
	shuffleOn: boolean,
): number | null {
	if (mode === "one") return i;
	if (shuffleOn) return pickRandom(songs, i);
	if (i <= 0) return mode === "all" ? songs.length - 1 : null;
	return i - 1;
}

export async function nextSong() {
	forward({ action: "next" });
	const q = queue.value;
	if (q.length === 0) return;
	const next = pickNext(q, currentIndex.value, repeat.value, shuffle.value);
	if (next == null) return;
	await playAtIndex(next);
}

export async function prevSong() {
	forward({ action: "prev" });
	const q = queue.value;
	if (q.length === 0) return;
	const prev = pickPrev(q, currentIndex.value, repeat.value, shuffle.value);
	if (prev == null) return;
	await playAtIndex(prev);
}

setOnTrackEnded(() => {
	if (!autoAdvance.value) return;
	const q = queue.value;
	if (q.length === 0) return;
	const next = pickNext(q, currentIndex.value, repeat.value, shuffle.value);
	if (next == null) return;
	void playAtIndex(next);
});

media.tracks({
	next: () => void nextSong(),
	previous: () => void prevSong(),
});
