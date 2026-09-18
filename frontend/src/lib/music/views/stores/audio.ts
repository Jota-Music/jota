import { effect, signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import { AudioCache } from "@/lib/music/views/stores/cache";
import * as media from "@/lib/music/views/stores/media-session";
import { setSongYoutubeId } from "@/lib/music/views/stores/queue";
import { publish } from "@/lib/music/views/stores/remote";
import { addError, logError } from "@/lib/shared/views/stores/errors";

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
// Identifies the latest play() call. A superseded call (a newer one started)
// must bail out without touching the shared element or reporting a failure.
let playSeq = 0;
// Counts running recovery ladders, so element errors are logged with detail
// instead of each surfacing in the error bar.
let recovering = 0;
// Set when WebKit rejects play() because the page has no user activation. That
// gates every track the same way, so it must stop the queue instead of failing
// each song in turn.
let blockedByPolicy = false;

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
// True from a track change until playback actually starts. A room keeps it set
// while it agrees on the new track; local playback never sets it.
export const pendingStart = signal(false);
export const isPlaying = signal(false);
export const progress = signal(0);
export const dragSeeking = signal(false);
export const audioDuration = signal(0);
export const volume = signal(getInitialVolume());
export const muted = signal(getInitialMuted());

export const currentSong = signal<Song | null>(null);

// Songs whose every recovery attempt failed, so the queue can flag them.
export const failedSongs = signal<Set<string>>(new Set());

function markFailed(id: string, failed: boolean): void {
	const next = new Set(failedSongs.value);
	if (failed) {
		next.add(id);
	} else {
		next.delete(id);
	}
	failedSongs.value = next;
}

// Apply a resolved YouTube ID to the song on screen and to its copy in the
// queue, so queue edits and replays keep it visible.
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

// What the media pipeline actually reported, so a failed load names its cause
// (bad source vs. decode vs. aborted) instead of a bare "play failed".
function mediaState(el: HTMLAudioElement | null): string {
	if (!el) return "element=none";
	const err = el.error;
	return [
		`readyState=${el.readyState}`,
		`networkState=${el.networkState}`,
		`paused=${el.paused}`,
		err
			? `mediaError=${err.code}(${err.message || "no message"})`
			: "mediaError=none",
	].join(" ");
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

// WebKit's autoplay policy rejects play() until the page has user activation.
// Not a broken track: retrying, rebuilding or re-resolving cannot change it.
function isNotAllowedError(err: unknown): boolean {
	return (
		typeof err === "object" &&
		err !== null &&
		(err as { name?: string }).name === "NotAllowedError"
	);
}

// "aborted" means a newer load superseded this one, which is not a failure and
// must not run the recovery ladder nor mark the song as failed.
type LoadStatus = "ok" | "failed" | "aborted";

// Same distinction for play(): a superseded call must not make the caller sweep
// to the next track as if the song had genuinely failed.
export type PlayResult = "ok" | "failed" | "aborted";

async function loadSongIntoPlayer(
	song: Song,
	autostart: boolean,
	startSeconds?: number | (() => number),
): Promise<LoadStatus> {
	const token = ++loadToken;

	stopEndWatch();
	endedElement = null;

	if (audio) {
		audio.pause();
		audio = null;
		loadedSongId = null;
		knownDuration = 0;
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
		if (token !== loadToken) return "aborted";
		isLoading.value = false;
		isPlaying.value = false;
		logError(err, `resolve ${songLabel(song)}`);
		return "failed";
	}

	let instance: HTMLAudioElement;
	try {
		instance = await AudioCache.getAudioElement(song);
	} catch (err) {
		if (token !== loadToken) return "aborted";
		isLoading.value = false;
		isPlaying.value = false;
		logError(err, `element ${songLabel(song)}`);
		return "failed";
	}
	if (token !== loadToken) return "aborted";

	audio = instance;
	loadedSongId = song.id;

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
				logError(err, `buffer ${songLabel(song)} | ${mediaState(instance)}`);
				return "failed";
			}
			const t = startAt();
			instance.currentTime = t;
			progress.value = t;
		}
		try {
			await instance.play();
			if (token !== loadToken) {
				instance.pause();
				return "aborted";
			}
			isPlaying.value = true;
		} catch (err) {
			if (token !== loadToken) return "aborted";
			if (!isAbortError(err)) {
				if (isNotAllowedError(err)) blockedByPolicy = true;
				isPlaying.value = false;
				isLoading.value = false;
				logError(err, `play ${songLabel(song)} | ${mediaState(instance)}`);
				return "failed";
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
			return "failed";
		}
		if (wantsStart) {
			const t = startAt();
			instance.currentTime = t;
			progress.value = t;
		}
	}

	if (token !== loadToken) return "aborted";
	isLoading.value = false;
	return "ok";
}

// Retry the element already loaded for this song, without rebuilding it or
// re-resolving the stream. The cheapest recovery step.
async function retryElement(song: Song): Promise<boolean> {
	if (
		!audio ||
		loadedSongId !== song.id ||
		audio.error ||
		endedElement === audio
	) {
		return false;
	}
	try {
		await audio.play();
		isPlaying.value = true;
		return true;
	} catch (err) {
		if (!isAbortError(err)) {
			if (isNotAllowedError(err)) blockedByPolicy = true;
			logError(err, `retry element ${songLabel(song)} | ${mediaState(audio)}`);
		}
		return false;
	}
}

// Play a song, escalating recovery from cheapest to costliest: reuse the cached
// stream, retry the loaded element, rebuild it, then ask the backend for a fresh
// stream (expired link, bad generation, bot check). Only when every step fails
// does the caller skip the track.
export async function play(
	song: Song,
	startSeconds?: number | (() => number),
): Promise<PlayResult> {
	const seq = ++playSeq;
	const superseded = () => seq !== playSeq;

	blockedByPolicy = false;

	const success = (): PlayResult => {
		markFailed(song.id, false);
		return "ok";
	};

	if (superseded()) return "aborted";

	// A consensus round already prepared this exact stream (the host waits for the
	// room, the guest warms it): resume it instead of rebuilding the pipeline and
	// re-buffering the same track.
	if (
		startSeconds == null &&
		loadedSongId === song.id &&
		audio &&
		!audio.error &&
		endedElement !== audio
	) {
		try {
			await audio.play();
			if (loadedSongId !== song.id) return "aborted";
			isPlaying.value = true;
			return success();
		} catch (err) {
			if (isNotAllowedError(err)) {
				blockedByPolicy = true;
				return blocked(song);
			}
			if (!isAbortError(err)) {
				logError(err, `resume ${songLabel(song)} | ${mediaState(audio)}`);
			}
		}
	}

	recovering++;
	try {
		const first = await loadSongIntoPlayer(song, true, startSeconds);
		if (first === "aborted" || superseded()) return "aborted";
		if (first === "ok") return success();
		if (blockedByPolicy) return blocked(song);

		if (await retryElement(song)) return success();
		if (superseded()) return "aborted";
		if (blockedByPolicy) return blocked(song);

		AudioCache.releaseElement(song.id);
		const second = await loadSongIntoPlayer(song, true, startSeconds);
		if (second === "aborted" || superseded()) return "aborted";
		if (second === "ok") return success();
		if (blockedByPolicy) return blocked(song);

		AudioCache.remove(song.id);
		const third = await loadSongIntoPlayer(song, true, startSeconds);
		if (third === "aborted" || superseded()) return "aborted";
		if (third === "ok") return success();
	} finally {
		recovering--;
	}

	if (superseded()) return "aborted";

	markFailed(song.id, true);
	addError(
		new Error("every recovery attempt failed"),
		`play "${song.name}" (${song.id})`,
	);
	return "failed";
}

// WebKit rejected play() for lack of user activation. The track is fine, so it
// must not be flagged or skipped; the caller stops and waits for an interaction.
function blocked(song: Song): "failed" {
	markFailed(song.id, false);
	// Not a broken track but a blocked one: release the room spinner so a host
	// keeps publishing state instead of freezing the whole room.
	pendingStart.value = false;
	armRecovery(song);
	addError(
		new Error("the browser needs a click to allow playback"),
		`play "${song.name}"`,
	);
	return "failed";
}

// Retry on the next user gesture. The retry runs synchronously inside the
// handler, so WebKit accepts it as gesture-initiated and grants the shared
// element permission for the rest of the session.
let recoveryArmed = false;

function armRecovery(song: Song): void {
	if (recoveryArmed || typeof window === "undefined") return;
	recoveryArmed = true;

	const retry = () => {
		window.removeEventListener("pointerdown", retry, true);
		window.removeEventListener("keydown", retry, true);
		recoveryArmed = false;

		if (audio && loadedSongId === song.id && !audio.error) {
			void audio.play().then(
				() => {
					isPlaying.value = true;
					markFailed(song.id, false);
				},
				() => {},
			);
			return;
		}
		void play(song);
	};

	window.addEventListener("pointerdown", retry, true);
	window.addEventListener("keydown", retry, true);
}

export function playbackBlocked(): boolean {
	return blockedByPolicy;
}

export async function prepareSong(
	song: Song,
	startSeconds?: number | (() => number),
): Promise<boolean> {
	return (await loadSongIntoPlayer(song, false, startSeconds)) === "ok";
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

function hasLoadedAudio(): boolean {
	return !!audio && !audio.error && endedElement !== audio;
}

// True when the element currently holds this exact song, so callers do not
// mistake a still-loaded previous track for the one the queue points at.
export function hasLoaded(songId: string): boolean {
	return loadedSongId === songId && hasLoadedAudio();
}

// Recover playback after a failed stream: reload the current song from a
// freshly resolved URL instead of reusing the dead element/URL.
export async function resume(): Promise<boolean> {
	if (audio && !audio.error && endedElement !== audio) {
		try {
			await audio.play();
			return true;
		} catch (err) {
			if (isAbortError(err)) return false;
			logError(
				err,
				`resume ${songLabel(currentSong.value)} | ${mediaState(audio)}`,
			);
		}
	}
	const song = currentSong.value;
	if (!song) return false;
	return (await play(song)) === "ok";
}

export function seek(time: number) {
	if (!Number.isFinite(time)) return;
	if (audio) {
		audio.currentTime = time;
	}
	progress.value = time;
}

// Seek and wait for the element to actually land, so a caller never starts
// playback from the old position while the seek is still in flight. Resolves on
// the `seeked` event, right away when already there, and on a timeout so a
// stream that never fires it cannot stall the caller.
export function seekTo(time: number, timeoutMs = 2000): Promise<void> {
	const el = audio;
	if (!el || !Number.isFinite(time)) return Promise.resolve();

	const target = clampSeconds(time, knownDurationOr(el.duration));
	if (Math.abs(el.currentTime - target) < 0.05) {
		seek(target);
		return Promise.resolve();
	}

	return new Promise<void>((resolve) => {
		let settled = false;
		const done = () => {
			if (settled) return;
			settled = true;
			clearTimeout(timer);
			el.removeEventListener("seeked", done);
			el.removeEventListener("error", done);
			resolve();
		};
		const timer = setTimeout(done, timeoutMs);
		el.addEventListener("seeked", done);
		el.addEventListener("error", done);
		seek(target);
	});
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

export function pause(): boolean {
	if (audio && !audio.error && endedElement !== audio && !audio.paused) {
		audio.pause();
		return true;
	}
	return false;
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
		audio.pause();
		audio = null;
		loadedSongId = null;
		knownDuration = 0;
	}
	currentSong.value = null;
	media.clear();
	progress.value = 0;
	audioDuration.value = 0;
	isPlaying.value = false;
	pendingStart.value = false;
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
		pendingStart.value = false;
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
		pendingStart.value = false;
		knownDuration = 0;
		if (loadedSongId) AudioCache.remove(loadedSongId);
		audio = null;
		loadedSongId = null;
		media.update(currentSong.value, false);
		const detail = `playback stopped ${label} | ${mediaState(a)}`;
		if (recovering > 0) {
			logError(new Error(`media error code ${code}`), detail);
		} else {
			addError(new Error(`media error code ${code}`), detail);
		}
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
	play: () => {
		publish({ action: "play" });
		void resume();
	},
	pause: () => {
		publish({ action: "pause" });
		pause();
	},
	stop: () => {
		publish({ action: "stop" });
		stopPlayer();
	},
	seek: (seconds) => {
		publish({ action: "seek", positionMs: Math.round(seconds * 1000) });
		seek(seconds);
	},
	position: getPlaybackSeconds,
});
