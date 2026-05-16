import { effect, signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import { AudioCache } from "@/lib/music/views/stores/cache";
import { isMainTab } from "@/lib/shared/views/ui/hooks/tabs";

const VOLUME_STORAGE_KEY = "audio-volume";
const MUTED_STORAGE_KEY = "audio-muted";

function parseStoredVolume(raw: string | null): number {
	if (raw === null) return 1;
	const parsed = Number(raw);
	if (Number.isFinite(parsed) && parsed >= 0 && parsed <= 1) {
		return parsed;
	}
	return 1;
}

function parseStoredMuted(raw: string | null): boolean {
	if (raw === null) return false;
	if (raw === "1" || raw === "true") return true;
	if (raw === "0" || raw === "false") return false;
	return false;
}

function checkTabMute(): boolean {
	return muted.value || !isMainTab.value;
}

/* -------------------------------------------------
   AUDIO INSTANCE
-------------------------------------------------- */

let audio: HTMLAudioElement | null = null;

let onTrackEndedCallback: (() => void) | null = null;

export function setOnTrackEnded(fn: () => void) {
	onTrackEndedCallback = fn;
}

let _broadcastToggle: (() => void) | null = null;

export function setBroadcastToggle(fn: (() => void) | null) {
	_broadcastToggle = fn;
}

/* -------------------------------------------------
   STATE
-------------------------------------------------- */

function getInitialVolume() {
	if (typeof window !== "undefined") {
		return parseStoredVolume(localStorage.getItem(VOLUME_STORAGE_KEY));
	}
	return 1;
}

function getInitialMuted() {
	if (typeof window !== "undefined") {
		return parseStoredMuted(localStorage.getItem(MUTED_STORAGE_KEY));
	}
	return false;
}

export const isLoading = signal(false);
export const isPlaying = signal(false);
export const progress = signal(0);
export const audioDuration = signal(0);
export const volume = signal(getInitialVolume());
export const muted = signal(getInitialMuted());

/* -------------------------------------------------
   CURRENT SONG (reference)
-------------------------------------------------- */

export const currentSong = signal<Song | null>(null);

/* -------------------------------------------------
   PLAYBACK
-------------------------------------------------- */

function waitUntilBufferedEnough(el: HTMLAudioElement): Promise<void> {
	if (el.readyState >= HTMLMediaElement.HAVE_ENOUGH_DATA) {
		return Promise.resolve();
	}
	return new Promise((resolve, reject) => {
		const done = () => {
			el.removeEventListener("canplaythrough", onReady);
			el.removeEventListener("canplay", onReady);
			el.removeEventListener("error", onErr);
		};
		const onReady = () => {
			done();
			resolve();
		};
		const onErr = () => {
			done();
			reject(new Error("audio load error"));
		};
		el.addEventListener("canplaythrough", onReady, { once: true });
		el.addEventListener("canplay", onReady, { once: true });
		el.addEventListener("error", onErr, { once: true });
	});
}

function clampStartSeconds(
	instance: HTMLAudioElement,
	seconds: number,
): number {
	if (!Number.isFinite(seconds) || seconds < 0) return 0;
	const d = instance.duration;
	if (Number.isFinite(d) && d > 0) {
		const eps = 0.05;
		return Math.min(seconds, Math.max(0, d - eps));
	}
	return seconds;
}

async function loadSongIntoPlayer(
	song: Song,
	autostart: boolean,
	startSeconds?: number,
) {
	currentSong.value = song;

	audioDuration.value = 0;

	isLoading.value = true;

	const data = await AudioCache.get(song);

	if (audio) {
		audio.pause();
		audio.src = "";
		audio.load();
		audio = null;
	}

	const instance = new Audio(data.url);
	audio = instance;

	instance.preload = "auto";
	instance.volume = volume.value;
	instance.muted = checkTabMute();
	instance.currentTime = 0;

	bindEvents(instance);

	const d0 = instance.duration;
	if (Number.isFinite(d0) && d0 > 0) {
		audioDuration.value = d0;
	}

	const wantsStart =
		startSeconds != null && Number.isFinite(startSeconds) && startSeconds > 0;

	if (autostart) {
		if (wantsStart) {
			await waitUntilBufferedEnough(instance);
			const t = clampStartSeconds(instance, startSeconds);
			instance.currentTime = t;
			progress.value = t;
		}
		try {
			await instance.play();
			isPlaying.value = true;
		} catch {
			isPlaying.value = false;
		}
	} else {
		isPlaying.value = false;
		await waitUntilBufferedEnough(instance);
		if (wantsStart) {
			const t = clampStartSeconds(instance, startSeconds);
			instance.currentTime = t;
			progress.value = t;
		}
	}

	isLoading.value = false;
}

export async function play(song: Song, startSeconds?: number) {
	await loadSongIntoPlayer(song, true, startSeconds);
}

export async function prepareSong(song: Song, startSeconds?: number) {
	await loadSongIntoPlayer(song, false, startSeconds);
}

export function getPlaybackSeconds(): number {
	if (audio && Number.isFinite(audio.currentTime)) {
		return Math.max(0, audio.currentTime);
	}
	const p = progress.value;
	return Number.isFinite(p) ? Math.max(0, p) : 0;
}

/* -------------------------------------------------
   CONTROLS
-------------------------------------------------- */

export function pause() {
	audio?.pause();
	_broadcastToggle?.();
}

function setPlayerPosition(seconds: number) {
	if (!Number.isFinite(seconds)) return;
	if (audio) {
		audio.currentTime = seconds;
	}
	progress.value = seconds;
}

export function seek(time: number) {
	setPlayerPosition(time);
}

export function socketSeek(time: number) {
	setPlayerPosition(time);
}

export function setVolume(value: number) {
	const v = Math.max(0, Math.min(1, value));
	volume.value = v;
	if (audio) audio.volume = v;
	if (typeof window !== "undefined") {
		try {
			localStorage.setItem(VOLUME_STORAGE_KEY, String(v));
		} catch {
			// ignore storage errors
		}
	}
}

export function setMuted(value: boolean) {
	muted.value = value;
	if (audio) audio.muted = checkTabMute();
	if (typeof window !== "undefined") {
		try {
			localStorage.setItem(MUTED_STORAGE_KEY, value ? "1" : "0");
		} catch {
			// ignore storage errors
		}
	}
}

export async function togglePlayPause(): Promise<boolean> {
	if (!audio) return false;
	if (audio.paused) {
		try {
			await audio.play();
		} catch {
			return false;
		}
	} else {
		audio.pause();
	}
	_broadcastToggle?.();
	return !audio.paused;
}

export async function remoteTogglePlayPause(): Promise<boolean> {
	if (!audio) return false;
	if (audio.paused) {
		try {
			await audio.play();
		} catch {
			return false;
		}
	} else {
		audio.pause();
	}
	return !audio.paused;
}

export function syncPlayerFromServer(playing: boolean) {
	if (audio) {
		if (playing) {
			void audio.play().catch(() => {});
		} else {
			audio.pause();
		}
	}
	isPlaying.value = playing;
}

export function stopPlayer() {
	if (audio) {
		audio.pause();
		audio.src = "";
		audio.load();
		audio = null;
	}
	currentSong.value = null;
	progress.value = 0;
	audioDuration.value = 0;
	isPlaying.value = false;
}

/* -------------------------------------------------
   EVENTS
-------------------------------------------------- */

function bindEvents(a: HTMLAudioElement) {
	const syncDuration = () => {
		const d = a.duration;
		audioDuration.value = Number.isFinite(d) && d > 0 ? d : 0;
	};

	a.addEventListener("loadedmetadata", syncDuration);
	a.addEventListener("durationchange", syncDuration);

	a.onplay = () => {
		isPlaying.value = true;
	};

	a.onpause = () => {
		isPlaying.value = false;
	};

	a.ontimeupdate = () => {
		progress.value = a.currentTime;
	};

	a.onended = () => {
		progress.value = 0;
		onTrackEndedCallback?.();
	};
}

export function toggleMute() {
	setMuted(!muted.value);
}

effect(() => {
	muted.value;
	isMainTab.value;
	if (audio) audio.muted = checkTabMute();
});

if (typeof window !== "undefined") {
	window.addEventListener("storage", (e: StorageEvent) => {
		if (e.storageArea !== localStorage) return;
		if (e.key === VOLUME_STORAGE_KEY) {
			const v = parseStoredVolume(e.newValue);
			volume.value = v;
			if (audio) audio.volume = v;
		} else if (e.key === MUTED_STORAGE_KEY) {
			muted.value = parseStoredMuted(e.newValue);
			if (audio) audio.muted = checkTabMute();
		}
	});
}
