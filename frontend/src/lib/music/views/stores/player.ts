import { pick } from "@/lib/music/app/playback";
import { pinAfter, sameQueue } from "@/lib/music/app/queue";
import { permute, random, seed } from "@/lib/music/app/shuffle";
import type { Song } from "@/lib/music/model";
import {
	audioDuration,
	clampSeconds,
	currentSong,
	dragSeeking,
	getPlaybackSeconds,
	hasLoaded,
	isPlaying,
	play,
	playbackBlocked,
	prepareSong,
	progress,
	seek,
	setOnTrackEnded,
	stopPlayer,
	togglePlayPause,
} from "@/lib/music/views/stores/audio";
import { AudioCache } from "@/lib/music/views/stores/cache";
import * as media from "@/lib/music/views/stores/media-session";
import {
	clampIndex,
	currentIndex,
	persistQueue,
	pingQueue,
	queue,
	queueSource,
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

export function previewSeek(seconds: number) {
	dragSeeking.value = true;
	progress.value = clampSeconds(seconds, audioDuration.value);
}

export function commitSeek(seconds: number) {
	dragSeeking.value = false;
	const clamped = clampSeconds(seconds, audioDuration.value);
	publish({ action: "seek", positionMs: clamped * 1000 });
	seek(clamped);
}

// setQueueState replaces the queue. source identifies the playlist/album it came
// from; undefined preserves the current source (reorders), null clears it.
function setQueueState(
	nextQueue: Song[],
	nextIndex: number,
	source: string | null | undefined = undefined,
) {
	if (source !== undefined) queueSource.value = source;

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

// Pick the next song in the queue that has not failed yet in this sweep,
// winding around from `from`. Returns null once every song has been tried, so a
// totally unavailable queue stops instead of looping forever.
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

	// play() exhausts this track's recovery ladder before returning "failed".
	const result = await play(song);
	if (queue.value[currentIndex.value]?.id !== song.id) return;

	// A newer play() superseded this one right after the track switched: nothing
	// failed, so do not sweep to the next song and let the newer load own it.
	if (result === "aborted") return;

	if (result === "failed") {
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
	source: string | null = null,
) {
	// Pressing play on the track that is already loaded restarts it instead of
	// reloading the same queue.
	if (isQueue(fullOrderedSongs) && currentSong.value?.id === clicked.id) {
		if (source !== null) queueSource.value = source;
		return restart();
	}

	const i = fullOrderedSongs.findIndex((s) => s.id === clicked.id);
	if (i === -1) return;

	const ordered = shuffle.value
		? permute(fullOrderedSongs, random(seed()))
		: fullOrderedSongs.slice();
	const index = ordered.findIndex((s) => s.id === clicked.id);
	await setQueueState(ordered, index, source);
	await playAtIndex(index);
}

export async function playAll(songs: Song[], source: string | null = null) {
	if (songs.length === 0) return;
	const ordered = shuffle.value
		? permute(songs, random(seed()))
		: songs.slice();
	await setQueueState(ordered, 0, source);
	await playAtIndex(0);
}

export function isQueue(songs: Song[]) {
	return sameQueue(queue.value, songs);
}

// A header play control starts the whole list unless that exact queue is already
// loaded. Membership alone would no-op on a cold start, when the queue is
// restored from storage but no track is loaded.
export function playList(songs: Song[], source: string | null = null) {
	if (isQueue(songs) && currentSong.value) {
		if (source !== null) queueSource.value = source;
		return toggleSong();
	}
	return playAll(songs, source);
}

let enqueueAnchor = -1;
let enqueueAnchorId: string | null = null;

export function enqueueMany(songs: Song[]) {
	if (songs.length === 0) return;

	// A position left over from a longer queue would insert at the end and stay
	// out of range, so reconcile it before it decides where the songs land.
	if (currentIndex.value < 0 || currentIndex.value >= queue.value.length) {
		currentIndex.value = clampIndex(currentIndex.value, queue.value.length);
	}

	const q = [...queue.value];
	const anchored =
		enqueueAnchorId != null && q[enqueueAnchor]?.id === enqueueAnchorId;
	const at = anchored
		? enqueueAnchor + 1
		: currentIndex.value >= 0 && currentIndex.value < q.length
			? currentIndex.value + 1
			: q.length;
	if (anchored && at > q.length) return;
	q.splice(at, 0, ...songs);
	enqueueAnchor = at + songs.length - 1;
	enqueueAnchorId = songs[songs.length - 1]?.id ?? null;
	queue.value = q;
	persistQueue();
	pingQueue();

	if (currentSong.value === null) {
		void playAtIndex(at);
		return;
	}

	if (currentIndex.value < 0) currentIndex.value = 0;
	preloadUpcomingSongs(q, currentIndex.value);
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
	const next = pinAfter(queue.value, currentIndex.value, from);
	if (!next) return;

	setQueueState(next.queue, next.index);
}

export async function playAt(i: number): Promise<void> {
	const q = queue.value;

	if (i < 0 || i >= q.length) return;
	if (i === currentIndex.value) {
		if (hasLoaded(q[i].id)) return restart();
		return;
	}

	await playAtIndex(i);
}

export async function toggleSong() {
	publish({ action: isPlaying.value ? "pause" : "play" });
	await togglePlayPause();
}

// Activating the track that is already loaded restarts it from the top: a row
// means "play this from the beginning". A paused track resumes from zero.
export async function restart(): Promise<void> {
	commitSeek(0);
	if (!isPlaying.value) await toggleSong();
}

// Reload the current track after its stream changed (a corrected YouTube id).
// In a room it must go through the load round so every member re-resolves it;
// locally it keeps the position and the play/pause state.
export async function reloadCurrent(): Promise<void> {
	const song = currentSong.value;
	if (!song) return;

	const gate = playGate.value;
	if (gate) {
		await gate(song);
		return;
	}

	const at = getPlaybackSeconds();
	if (isPlaying.value) {
		await play(song, at);
	} else {
		await prepareSong(song, at);
	}
}

// Pressing next/previous goes straight to the load. A debounce here pushed
// play() past the user gesture, so WebKit rejected it and the queue sat silent
// until the next click. Superseded loads are already dropped by the play
// sequence, so rapid presses need no timer of their own.
export async function nextSong() {
	const q = queue.value;
	if (q.length === 0) return;
	const next = pick(q, currentIndex.value, repeat.value, 1);
	if (next == null) return;
	await playAtIndex(next);
}

export async function prevSong() {
	const q = queue.value;
	if (q.length === 0) return;
	const prev = pick(q, currentIndex.value, repeat.value, -1);
	if (prev == null) return;
	await playAtIndex(prev);
}

setOnTrackEnded(() => {
	const q = queue.value;
	if (q.length === 0) return;
	const next = pick(q, currentIndex.value, repeat.value, 1);
	if (next == null) return;
	void playAtIndex(next);
});

media.tracks({
	next: () => void nextSong(),
	previous: () => void prevSong(),
});
