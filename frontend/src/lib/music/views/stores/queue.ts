import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import { forward } from "@/lib/music/views/stores/remote";

const QUEUE_STORAGE_KEY = "music-queue";
const INDEX_STORAGE_KEY = "music-queue-index";
const SHUFFLE_STORAGE_KEY = "music-shuffle";
const REPEAT_STORAGE_KEY = "music-repeat";

type RepeatMode = "off" | "all" | "one";

function loadQueue(): Song[] {
	if (typeof window === "undefined") return [];
	try {
		const raw = localStorage.getItem(QUEUE_STORAGE_KEY);
		if (!raw) return [];
		const parsed = JSON.parse(raw);
		return Array.isArray(parsed) ? parsed : [];
	} catch {
		return [];
	}
}

function loadIndex(): number {
	if (typeof window === "undefined") return -1;
	try {
		const raw = localStorage.getItem(INDEX_STORAGE_KEY);
		if (!raw) return -1;
		const parsed = Number(raw);
		return Number.isFinite(parsed) ? parsed : -1;
	} catch {
		return -1;
	}
}

function loadShuffle(): boolean {
	if (typeof window === "undefined") return false;
	try {
		const raw = localStorage.getItem(SHUFFLE_STORAGE_KEY);
		return raw === "1";
	} catch {
		return false;
	}
}

function loadRepeat(): RepeatMode {
	if (typeof window === "undefined") return "off";
	try {
		const raw = localStorage.getItem(REPEAT_STORAGE_KEY);
		return raw === "all" || raw === "one" ? raw : "off";
	} catch {
		return "off";
	}
}

function saveQueue(q: Song[]) {
	if (typeof window === "undefined") return;
	try {
		localStorage.setItem(QUEUE_STORAGE_KEY, JSON.stringify(q));
	} catch {}
}

function saveIndex(i: number) {
	if (typeof window === "undefined") return;
	try {
		localStorage.setItem(INDEX_STORAGE_KEY, String(i));
	} catch {}
}

function saveShuffle(v: boolean) {
	if (typeof window === "undefined") return;
	try {
		localStorage.setItem(SHUFFLE_STORAGE_KEY, v ? "1" : "0");
	} catch {}
}

function saveRepeat(v: RepeatMode) {
	if (typeof window === "undefined") return;
	try {
		localStorage.setItem(REPEAT_STORAGE_KEY, v);
	} catch {}
}

export const queue = signal<Song[]>(loadQueue());

export const currentIndex = signal<number>(loadIndex());

export const shuffle = signal<boolean>(loadShuffle());

export const repeat = signal<RepeatMode>(loadRepeat());

export const playerSkeletonOn = signal(false);

export const showQueue = signal<boolean>(false);

export function persistQueue() {
	saveQueue(queue.value);
	saveIndex(currentIndex.value);
}

export function clearQueue() {
	queue.value = [];
	currentIndex.value = -1;
	saveQueue([]);
	saveIndex(-1);
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
