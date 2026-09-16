import { effect, signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import { AudioCache } from "@/lib/music/views/stores/cache";
import * as media from "@/lib/music/views/stores/media-session";
import { setSongYoutubeId } from "@/lib/music/views/stores/queue";
import { addError } from "@/lib/shared/views/stores/errors";

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
	return false;
}

let audio: HTMLAudioElement | null = null;
let loadedSongId: string | null = null;
let knownDuration = 0;
let loadToken = 0;

let onTrackEndedCallback: (() => void) | null = null;
let endedElement: HTMLAudioElement | null = null;
let endWatch: ReturnType<typeof setInterval> | null = null;

export function setOnTrackEnded(fn: () => void) {
	onTrackEndedCallback = fn;
}

function stopEndWatch() {
	if (endWatch !== null) {
		clearInterval(endWatch);
		endWatch = null;
	}
}

function endPlayback(a: HTMLAudioElement) {
	if (audio !== a || endedElement === a) return;
	endedElement = a;
	stopEndWatch();
	a.pause();
	progress.value = 0;
	isPlaying.value = false;
	media.update(currentSong.value, false);
	onTrackEndedCallback?.();
}

// WebKit on macOS does not always fire `ended` for YouTube CDN streams, so end
// the track once it reaches its known duration or stalls right at the end.
function watchEnd(a: HTMLAudioElement) {
	stopEndWatch();
	let last = -1;
	let still = 0;
	endWatch = setInterval(() => {
		if (audio !== a || a.ended) {
			stopEndWatch();
			return;
		}
		if (a.paused) {
			last = -1;
			still = 0;
			return;
		}
		const elementDuration = a.duration;
		const fromElement = Number.isFinite(elementDuration) && elementDuration > 0;
		const time = a.currentTime;
		if (time === last) {
			still += 250;
		} else {
			still = 0;
			last = time;
		}
		// `knownDuration` is truncated from the URL, so only use it as a
		// near-end gate; the element duration triggers the end itself.
		const gate = knownDuration > 0 ? knownDuration : elementDuration;
		const reached = fromElement && time >= elementDuration - 0.25;
		const stalled =
			Number.isFinite(gate) && time > 0 && time >= gate - 2 && still >= 1000;
		if (reached || stalled) endPlayback(a);
	}, 250);
}

export const isLoading = signal(false);
export const isPlaying = signal(false);
export const progress = signal(0);
export const dragSeeking = signal(false);
export const seekCount = signal(0);
export const audioDuration = signal(0);
export const volume = signal(getInitialVolume());
export const muted = signal(getInitialMuted());

export const currentSong = signal<Song | null>(null);

// Apply a resolved YouTube ID to the song on screen and to its queued copy, so
// queue edits and replays keep it visible.
export function setYoutube(song: Song, youtube: string) {
	const current = currentSong.value;
	if (current && current.id === song.id && current.youtubeId !== youtube) {
		currentSong.value = { ...current, youtubeId: youtube };
	}

	setSongYoutubeId(song.id, youtube);
}

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

function waitForPlayable(
	el: HTMLAudioElement,
	timeoutMs = 15000,
): Promise<boolean> {
	if (el.readyState >= HTMLMediaElement.HAVE_ENOUGH_DATA) {
		return Promise.resolve(true);
	}
	return new Promise((resolve) => {
		const timer = setTimeout(() => {
			done();
			resolve(false);
		}, timeoutMs);
		const done = () => {
			clearTimeout(timer);
			el.removeEventListener("canplaythrough", onReady);
			el.removeEventListener("canplay", onReady);
			el.removeEventListener("error", onErr);
		};
		const onReady = () => {
			done();
			resolve(true);
		};
		const onErr = () => {
			done();
			resolve(false);
		};
		el.addEventListener("canplaythrough", onReady, { once: true });
		el.addEventListener("canplay", onReady, { once: true });
		el.addEventListener("error", onErr, { once: true });
	});
}

async function waitUntilBufferedEnough(el: HTMLAudioElement): Promise<void> {
	if (await waitForPlayable(el)) return;
	throw new Error(
		`audio load error (code ${el.error?.code ?? 0}, readyState ${el.readyState})`,
	);
}

// Identifies the track in error reports: the Spotify id plus the YouTube id
// when known, which is what actually explains a failed resolve or stream.
function songLabel(song: Song | null): string {
	if (!song) return "unknown song";
	return song.youtubeId ? `${song.id} (yt ${song.youtubeId})` : song.id;
}

function knownDurationOr(d: number): number {
	return knownDuration > 0 ? knownDuration : d;
}

export function clampSeconds(seconds: number, duration: number): number {
	if (!Number.isFinite(seconds) || seconds < 0) return 0;
	if (Number.isFinite(duration) && duration > 0) {
		const eps = 0.05;
		return Math.min(seconds, Math.max(0, duration - eps));
	}
	return seconds;
}

function isAbortError(err: unknown): boolean {
	return (
		typeof DOMException !== "undefined" &&
		err instanceof DOMException &&
		err.name === "AbortError"
	);
}

async function loadSongIntoPlayer(
	song: Song,
	autostart: boolean,
	startSeconds?: number | (() => number),
): Promise<boolean> {
	const token = ++loadToken;

	stopEndWatch();
	endedElement = null;

	if (audio) {
		const prevId = loadedSongId;
		audio.pause();
		audio.src = "";
		audio.load();
		audio = null;
		loadedSongId = null;
		knownDuration = 0;
		AudioCache.pin(null);
		if (prevId) AudioCache.releaseElement(prevId);
	}

	progress.value = 0;
	audioDuration.value = 0;
	isLoading.value = true;

	currentSong.value = song;
	media.update(song, isPlaying.value);

	let data: { url: string; youtube: string; duration: number };
	try {
		data = await AudioCache.get(song);
		knownDuration = data.duration > 0 ? data.duration : 0;
		if (data.youtube) setYoutube(song, data.youtube);
	} catch (err) {
		if (token !== loadToken) return false;
		isLoading.value = false;
		isPlaying.value = false;
		addError(err, `resolve audio ${songLabel(song)}`);
		return false;
	}

	let instance: HTMLAudioElement;
	try {
		instance = await AudioCache.getAudioElement(song);
	} catch (err) {
		if (token !== loadToken) return false;
		isLoading.value = false;
		isPlaying.value = false;
		addError(err, `audio element ${songLabel(song)}`);
		return false;
	}
	if (token !== loadToken) {
		AudioCache.releaseElement(song.id);
		return false;
	}

	audio = instance;
	loadedSongId = song.id;
	AudioCache.pin(song.id);

	instance.volume = volume.value;
	instance.muted = muted.value;
	instance.currentTime = 0;

	bindEvents(instance);

	const d0 = knownDurationOr(instance.duration);
	if (Number.isFinite(d0) && d0 > 0) {
		audioDuration.value = d0;
	}

	const wantsStart = startSeconds != null;
	const startAt = (): number => {
		const raw =
			typeof startSeconds === "function" ? startSeconds() : startSeconds;
		return clampSeconds(raw ?? 0, knownDurationOr(instance.duration));
	};

	if (autostart) {
		if (wantsStart) {
			try {
				await waitUntilBufferedEnough(instance);
			} catch (err) {
				isLoading.value = false;
				isPlaying.value = false;
				addError(err, `buffer ${songLabel(song)}`);
				return false;
			}
			const t = startAt();
			instance.currentTime = t;
			progress.value = t;
		}
		try {
			await instance.play();
			if (token !== loadToken) {
				instance.pause();
				return false;
			}
			isPlaying.value = true;
		} catch (err) {
			if (token !== loadToken) return false;
			if (!isAbortError(err)) {
				isPlaying.value = false;
				isLoading.value = false;
				addError(err, `play ${songLabel(song)}`);
				return false;
			}
		}
	} else {
		isPlaying.value = false;
		try {
			await waitUntilBufferedEnough(instance);
		} catch (err) {
			isLoading.value = false;
			isPlaying.value = false;
			addError(err, `buffer ${songLabel(song)}`);
			return false;
		}
		if (wantsStart) {
			const t = startAt();
			instance.currentTime = t;
			progress.value = t;
		}
	}

	if (token !== loadToken) return false;
	isLoading.value = false;
	return true;
}

export async function play(
	song: Song,
	startSeconds?: number | (() => number),
): Promise<boolean> {
	return await loadSongIntoPlayer(song, true, startSeconds);
}

export async function prepareSong(
	song: Song,
	startSeconds?: number | (() => number),
): Promise<boolean> {
	return await loadSongIntoPlayer(song, false, startSeconds);
}

export async function warm(song: Song, timeoutMs = 6000): Promise<boolean> {
	let el: HTMLAudioElement;
	try {
		el = await AudioCache.getAudioElement(song);
	} catch {
		return false;
	}
	if (el.readyState >= HTMLMediaElement.HAVE_ENOUGH_DATA) return true;

	return await Promise.race([
		waitForPlayable(el),
		new Promise<boolean>((resolve) => {
			setTimeout(
				() => resolve(el.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA),
				timeoutMs,
			);
		}),
	]);
}

export function getPlaybackSeconds(): number {
	if (
		audio &&
		loadedSongId === currentSong.value?.id &&
		Number.isFinite(audio.currentTime)
	) {
		return Math.max(0, audio.currentTime);
	}
	return 0;
}

export function hasLoadedAudio(): boolean {
	return !!audio && !audio.error && endedElement !== audio;
}

export function pause() {
	audio?.pause();
}

// Recover playback after a failed stream: reload the current song from a
// freshly resolved URL instead of reusing the dead element/URL.
async function resume(): Promise<boolean> {
	if (audio && !audio.error && endedElement !== audio) {
		try {
			await audio.play();
			return true;
		} catch (err) {
			if (isAbortError(err)) return false;
			addError(err, `resume ${songLabel(currentSong.value)}`);
			return false;
		}
	}
	const song = currentSong.value;
	if (!song) return false;
	AudioCache.remove(song.id);
	return await play(song);
}

export function seek(time: number) {
	if (!Number.isFinite(time)) return;
	if (audio) {
		audio.currentTime = time;
	}
	progress.value = time;
	seekCount.value++;
}

export function setPlaybackRate(rate: number) {
	if (audio) audio.playbackRate = rate;
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

function setMuted(value: boolean) {
	muted.value = value;
	if (audio) audio.muted = muted.value;
	if (typeof window !== "undefined") {
		try {
			localStorage.setItem(MUTED_STORAGE_KEY, value ? "1" : "0");
		} catch {
			// ignore storage errors
		}
	}
}

export async function togglePlayPause(): Promise<boolean> {
	if (audio && !audio.error && endedElement !== audio && !audio.paused) {
		audio.pause();
		return false;
	}
	return await resume();
}

export function stopPlayer() {
	loadToken++;
	stopEndWatch();
	endedElement = null;
	if (audio) {
		const prevId = loadedSongId;
		audio.pause();
		audio.src = "";
		audio.load();
		audio = null;
		loadedSongId = null;
		knownDuration = 0;
		if (prevId) AudioCache.releaseElement(prevId);
	}
	AudioCache.pin(null);
	currentSong.value = null;
	media.clear();
	progress.value = 0;
	audioDuration.value = 0;
	isPlaying.value = false;
}

function bindEvents(a: HTMLAudioElement) {
	const syncDuration = () => {
		if (audio !== a) return;
		const d = knownDurationOr(a.duration);
		audioDuration.value = Number.isFinite(d) && d > 0 ? d : 0;
	};

	a.onloadedmetadata = syncDuration;
	a.ondurationchange = syncDuration;

	a.onplay = () => {
		if (audio !== a) return;
		endedElement = null;
		isPlaying.value = true;
		media.update(currentSong.value, true);
		watchEnd(a);
	};

	a.onpause = () => {
		if (audio !== a) return;
		isPlaying.value = false;
		media.update(currentSong.value, false);
		stopEndWatch();
	};

	a.ontimeupdate = () => {
		if (audio !== a) return;
		if (!dragSeeking.value) progress.value = a.currentTime;
		media.position(a, knownDuration);
	};

	a.onended = () => {
		endPlayback(a);
	};

	a.onerror = () => {
		if (audio !== a) return;
		const code = a.error?.code ?? 0;
		const label = songLabel(currentSong.value);
		a.onerror = null;
		stopEndWatch();
		isPlaying.value = false;
		isLoading.value = false;
		knownDuration = 0;
		if (loadedSongId) AudioCache.remove(loadedSongId);
		audio = null;
		loadedSongId = null;
		AudioCache.pin(null);
		media.update(currentSong.value, false);
		addError(
			new Error(`media error code ${code}`),
			`playback stopped ${label}`,
		);
	};
}

export function toggleMute() {
	setMuted(!muted.value);
}

effect(() => {
	muted.value;
	if (audio) audio.muted = muted.value;
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
			if (audio) audio.muted = muted.value;
		}
	});
}

media.setup({
	play: () => void resume(),
	pause: () => audio?.pause(),
	stop: stopPlayer,
	seek,
	position: getPlaybackSeconds,
});
