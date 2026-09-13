import type { Song } from "@/lib/music/model";
import {
	audioDuration,
	currentSong,
	play,
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

function clampPlaybackSeconds(seconds: number): number {
	if (!Number.isFinite(seconds) || seconds < 0) return 0;
	const d = audioDuration.value;
	if (Number.isFinite(d) && d > 0) {
		const eps = 0.05;
		return Math.min(seconds, Math.max(0, d - eps));
	}
	return seconds;
}

function preloadUpcomingSongs(songs: Song[], idx: number) {
	const upcoming = songs.slice(idx + 1, idx + 3);
	if (upcoming.length > 0) void AudioCache.preload(...upcoming);
}

export function seekFromLocalControl(seconds: number) {
	seek(clampPlaybackSeconds(seconds));
}

function setQueueState(nextQueue: Song[], nextIndex: number) {
	const prevId = currentSong.value?.id ?? null;
	const song =
		nextIndex >= 0 && nextIndex < nextQueue.length
			? nextQueue[nextIndex]
			: null;
	const sameTrack = prevId != null && song != null && prevId === song.id;

	queue.value = nextQueue;
	currentIndex.value = nextIndex;
	persistQueue();

	if (!song) {
		stopPlayer();
		return;
	}

	if (sameTrack) {
		currentSong.value = song;
		preloadUpcomingSongs(nextQueue, nextIndex);
		return;
	}

	currentSong.value = song;
	preloadUpcomingSongs(nextQueue, nextIndex);
}

async function playAtIndex(i: number): Promise<void> {
	const q = queue.value;
	if (i < 0 || i >= q.length) return;

	currentIndex.value = i;
	persistQueue();
	const song = q[i];

	const ok = await play(song);
	if (!ok) {
		stopPlayer();
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
	if (i === currentIndex.value) return;

	await playAtIndex(i);
}

export async function toggleSong() {
	await togglePlayPause();
}

export async function nextSong() {
	const q = queue.value;
	const i = currentIndex.value;
	const r = repeat.value;
	const s = shuffle.value;

	if (q.length === 0) return;

	if (r === "one") {
		await playAtIndex(i);
		return;
	}

	let nextIndex: number;

	if (s) {
		const candidates = q.filter((_, idx) => idx !== i);
		if (candidates.length === 0) return;
		const random = candidates[Math.floor(Math.random() * candidates.length)];
		nextIndex = q.findIndex((song) => song.id === random.id);
	} else if (i >= q.length - 1) {
		if (r === "all") {
			nextIndex = 0;
		} else {
			return;
		}
	} else {
		nextIndex = i + 1;
	}

	await playAtIndex(nextIndex);
}

export async function prevSong() {
	const q = queue.value;
	const i = currentIndex.value;
	const r = repeat.value;
	const s = shuffle.value;

	if (q.length === 0) return;

	if (r === "one") {
		await playAtIndex(i);
		return;
	}

	let prevIndex: number;

	if (s) {
		const candidates = q.filter((_, idx) => idx !== i);
		if (candidates.length === 0) return;
		const random = candidates[Math.floor(Math.random() * candidates.length)];
		prevIndex = q.findIndex((song) => song.id === random.id);
	} else if (i <= 0) {
		if (r === "all") {
			prevIndex = q.length - 1;
		} else {
			return;
		}
	} else {
		prevIndex = i - 1;
	}

	await playAtIndex(prevIndex);
}

setOnTrackEnded(() => {
	const q = queue.value;
	const i = currentIndex.value;
	const r = repeat.value;
	const s = shuffle.value;

	if (q.length === 0) return;

	if (r === "one") {
		void playAtIndex(i);
		return;
	}

	let nextIndex: number;

	if (s) {
		const candidates = q.filter((_, idx) => idx !== i);
		if (candidates.length === 0) return;
		const random = candidates[Math.floor(Math.random() * candidates.length)];
		nextIndex = q.findIndex((song) => song.id === random.id);
	} else if (i >= q.length - 1) {
		if (r === "all") {
			nextIndex = 0;
		} else {
			return;
		}
	} else {
		nextIndex = i + 1;
	}

	void playAtIndex(nextIndex);
});

media.tracks({
	next: () => void nextSong(),
	previous: () => void prevSong(),
});
