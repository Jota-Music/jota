import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import { forward } from "@/lib/music/views/stores/remote";

const QUEUE_STORAGE_KEY = "music-queue";
const INDEX_STORAGE_KEY = "music-queue-index";
const SHUFFLE_STORAGE_KEY = "music-shuffle";
const REPEAT_STORAGE_KEY = "music-repeat";

type RepeatMode = "off" | "all" | "one";

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
		return Array.isArray(parsed) ? parsed : [];
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

export const shuffle = signal<boolean>(loadShuffle());

export const repeat = signal<RepeatMode>(loadRepeat());

export const showQueue = signal<boolean>(false);

export function persistQueue() {
	saveQueue(queue.value);
	saveIndex(currentIndex.value);
}

export function toggleShuffle() {
	const next = !shuffle.value;
	forward({ action: "shuffle" });
	shuffle.value = next;
	saveShuffle(next);
}

export function cycleRepeat() {
	const modes: RepeatMode[] = ["off", "all", "one"];
	const current = repeat.value;
	const next = modes[(modes.indexOf(current) + 1) % modes.length];
	forward({ action: "repeat" });
	repeat.value = next;
	saveRepeat(next);
}

export function setShuffle(v: boolean) {
	shuffle.value = v;
	saveShuffle(v);
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
}
