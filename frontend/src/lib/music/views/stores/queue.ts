import { computed, signal } from "@preact/signals";
import { sameQueue } from "@/lib/music/app/queue";
import { permute, random, seed } from "@/lib/music/app/shuffle";
import type { RepeatMode, Song } from "@/lib/music/model";
import { publish } from "@/lib/music/views/stores/remote";
import { get as read, set as write } from "@/lib/shared/utils/storage";

const QUEUE_STORAGE_KEY = "music-queue";
const INDEX_STORAGE_KEY = "music-queue-index";
const SHUFFLE_STORAGE_KEY = "music-shuffle";
const REPEAT_STORAGE_KEY = "music-repeat";
const QUEUE_SOURCE_STORAGE_KEY = "music-queue-source";
const HEARD_STORAGE_KEY = "music-queue-heard";
const HEARD_LIMIT = 100;

function loadQueue(): Song[] {
	const raw = read(QUEUE_STORAGE_KEY);
	if (!raw) return [];
	try {
		const parsed = JSON.parse(raw);
		return Array.isArray(parsed) ? (parsed as Song[]) : [];
	} catch {
		return [];
	}
}

function loadIndex(): number {
	const raw = read(INDEX_STORAGE_KEY);
	if (!raw) return -1;
	const parsed = Number(raw);
	return Number.isFinite(parsed) ? parsed : -1;
}

function loadShuffle(): boolean {
	return read(SHUFFLE_STORAGE_KEY) === "1";
}

function loadRepeat(): RepeatMode {
	const raw = read(REPEAT_STORAGE_KEY);
	return raw === "all" || raw === "one" ? raw : "off";
}

function saveQueue(q: Song[]) {
	write(QUEUE_STORAGE_KEY, JSON.stringify(q));
}

function saveIndex(i: number) {
	write(INDEX_STORAGE_KEY, String(i));
}

function saveShuffle(v: boolean) {
	write(SHUFFLE_STORAGE_KEY, v ? "1" : "0");
}

function saveRepeat(v: RepeatMode) {
	write(REPEAT_STORAGE_KEY, v);
}

function loadQueueSource(): string | null {
	return read(QUEUE_SOURCE_STORAGE_KEY) || null;
}

function saveQueueSource(v: string | null) {
	write(QUEUE_SOURCE_STORAGE_KEY, v ?? "");
}

function loadHeard(): string[] {
	const raw = read(HEARD_STORAGE_KEY);
	if (!raw) return [];
	try {
		const parsed = JSON.parse(raw);
		return Array.isArray(parsed)
			? parsed.filter((id): id is string => typeof id === "string")
			: [];
	} catch {
		return [];
	}
}

function saveHeard() {
	write(HEARD_STORAGE_KEY, JSON.stringify(heard));
}

// The index may outlive the queue it points into: a room snapshot replaces the
// queue with a shorter one, and a stale index persists in storage across
// reloads. An out-of-range index breaks pick(), leaves the panel with nothing to
// render and sends every enqueue to the end, so it is clamped on every load and
// replacement.
export function clampIndex(index: number, length: number): number {
	if (length <= 0 || index < 0) return -1;
	return Math.min(index, length - 1);
}

export const queue = signal<Song[]>(loadQueue());

export const currentIndex = signal<number>(
	clampIndex(loadIndex(), queue.value.length),
);

// Skip forward availability as a computed, so the button can subscribe to it
// directly. Reading queue.value for the length instead would subscribe the whole
// player to every queue edit (an added song re-renders it entirely).
// Repeat on: skipping always has a target (next wraps to the other end), so the
// button stays enabled even at the queue boundary and matches what nextSong
// actually does. There is no matching computed for back: a press restarts the
// track, which is available wherever one is loaded.
export const canNext = computed(() => {
	const i = currentIndex.value;
	const n = queue.value.length;
	if (i < 0 || i >= n) return false;
	return repeat.value !== "off" || i < n - 1;
});

// The playlist or album the current queue was started from, when known. Lets a
// card's play button reflect (and toggle) its own playback. Restored like the
// queue, so the association survives a reload.
export const queueSource = signal<string | null>(loadQueueSource());

export const shuffle = signal<boolean>(loadShuffle());

export const repeat = signal<RepeatMode>(loadRepeat());

export const showQueue = signal<boolean>(false);

export const queuePulse = signal(0);

export function pingQueue() {
	queuePulse.value++;
}

// Swap the whole queue (a room snapshot) and keep the position on the same
// track: `anchorId` is the playing song, re-found in the new order so an insert
// ahead of it does not slide the highlight onto a different row. Without an
// anchor the position is only clamped into range.
export function replaceQueue(songs: Song[], anchorId?: string | null) {
	queue.value = songs;
	const at = anchorId ? songs.findIndex((song) => song.id === anchorId) : -1;
	currentIndex.value =
		at !== -1 ? at : clampIndex(currentIndex.value, songs.length);
	persistQueue();
}

// The trail of tracks this queue actually played, newest last. A shuffled queue
// holds songs before the current one that were never played, so going back reads
// this instead of the array order.
let heard: string[] = loadHeard();

export function remember(song: Song) {
	if (heard[heard.length - 1] === song.id) return;
	heard.push(song.id);
	if (heard.length > HEARD_LIMIT) heard.shift();
	persistQueue();
}

// Index of the track played before the current one, or null when this queue has
// nothing behind the playhead. Entries the queue no longer holds are dropped.
export function unwind(songs: Song[]): number | null {
	let found: number | null = null;
	while (heard.length > 0) {
		const id = heard.pop();
		const at = songs.findIndex((song) => song.id === id);
		if (at !== -1) {
			found = at;
			break;
		}
	}
	// The entries popped are consumed, and the trail outlives a reload: leaving
	// them in storage would walk back through songs already played.
	persistQueue();
	return found;
}

export function clearHeard() {
	heard = [];
	persistQueue();
}

// A new list starts a new trail: what the previous queue played says nothing
// about this one. Tapping another track in the list that is already playing
// keeps it, so back still returns the song the user heard.
export function startTrail(next: Song[]) {
	if (sameQueue(queue.value, next)) return;
	clearHeard();
}

// Writes are debounced: play/skip/reorder mutate the queue several times a
// second, and each synchronous localStorage write freezes the main thread. The
// last state wins after a quiet pause, flushed immediately on hide or close so
// nothing is lost.
const QUEUE_SAVE_DELAY = 500;
let saveTimer: number | null = null;
let saveDirty = false;

export function persistQueue() {
	saveDirty = true;
	if (saveTimer != null) return;
	if (typeof window === "undefined") {
		flushQueue();
		return;
	}
	saveTimer = window.setTimeout(() => {
		saveTimer = null;
		flushQueue();
	}, QUEUE_SAVE_DELAY);
}

function flushQueue() {
	if (!saveDirty) return;
	saveDirty = false;
	saveQueue(queue.value);
	saveIndex(currentIndex.value);
	saveQueueSource(queueSource.value);
	saveHeard();
}

export function toggleShuffle() {
	if (shuffle.value) {
		applyShuffle(false);
		publish({ action: "shuffle", on: false });
		return;
	}
	const s = seed();
	applyShuffle(true, s);
	publish({ action: "shuffle", on: true, seed: s });
}

// Reorders the queue for the announced seed. A peer applies the sender's seed
// instead of drawing its own, so every member plays the same sequence.
export function applyShuffle(on: boolean, s?: number) {
	shuffle.value = on;
	saveShuffle(on);
	if (!on || s == null) return;

	const anchor = queue.value[currentIndex.value];
	const ordered = permute(queue.value, random(s));
	queue.value = ordered;
	if (anchor) {
		const idx = ordered.findIndex((song) => song.id === anchor.id);
		if (idx !== -1) currentIndex.value = idx;
	}
	persistQueue();
}

const REPEAT_MODES: RepeatMode[] = ["off", "all", "one"];

export function nextRepeat(mode: RepeatMode): RepeatMode {
	return REPEAT_MODES[(REPEAT_MODES.indexOf(mode) + 1) % REPEAT_MODES.length];
}

export function cycleRepeat() {
	const mode = nextRepeat(repeat.value);
	setRepeat(mode);
	publish({ action: "repeat", mode });
}

export function setRepeat(v: RepeatMode) {
	repeat.value = v;
	saveRepeat(v);
}

export function setSongYoutubeId(id: string, youtubeId: string) {
	const items = queue.value;
	const idx = items.findIndex((song) => song.id === id);
	if (idx === -1 || items[idx].youtubeId === youtubeId) return;

	const next = items.slice();
	next[idx] = { ...next[idx], youtubeId };
	queue.value = next;
	persistQueue();
}

if (typeof window !== "undefined") {
	const flush = () => {
		if (saveTimer != null) {
			clearTimeout(saveTimer);
			saveTimer = null;
		}
		flushQueue();
	};
	window.addEventListener("beforeunload", flush);
	window.addEventListener("pagehide", flush);
	document.addEventListener("visibilitychange", () => {
		if (document.hidden) flush();
	});
}
