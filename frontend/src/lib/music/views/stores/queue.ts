import { signal } from "@preact/signals";
import { permute, random, seed } from "@/lib/music/app/shuffle";
import type { RepeatMode, Song } from "@/lib/music/model";
import { publish } from "@/lib/music/views/stores/remote";

const QUEUE_STORAGE_KEY = "music-queue";
const INDEX_STORAGE_KEY = "music-queue-index";
const SHUFFLE_STORAGE_KEY = "music-shuffle";
const REPEAT_STORAGE_KEY = "music-repeat";

function read(key: string): string | null {
	if (typeof window === "undefined") return null;
	try {
		return localStorage.getItem(key);
	} catch {
		return null;
	}
}

function write(key: string, value: string): void {
	if (typeof window === "undefined") return;
	try {
		localStorage.setItem(key, value);
	} catch {}
}

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

export const queue = signal<Song[]>(loadQueue());

export const currentIndex = signal<number>(loadIndex());

// The playlist or album the current queue was started from, when known. Lets a
// card's play button reflect (and toggle) its own playback.
export const queueSource = signal<string | null>(null);

export const shuffle = signal<boolean>(loadShuffle());

export const repeat = signal<RepeatMode>(loadRepeat());

export const showQueue = signal<boolean>(false);

export const queuePulse = signal(0);

export function pingQueue() {
	queuePulse.value++;
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
	document.addEventListener("visibilitychange", () => {
		if (document.hidden) flush();
	});
}
