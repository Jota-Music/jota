import { effect, signal } from "@preact/signals";
import { shouldEnd } from "@/lib/music/app/end";
import {
	playing,
	RESUME_MS,
	retryDelay,
	stalled,
	tick,
} from "@/lib/music/app/recovery";
import { gain } from "@/lib/music/app/volume";
import type { Song } from "@/lib/music/model";
import { AudioCache } from "@/lib/music/views/stores/cache";
import * as media from "@/lib/music/views/stores/media-session";
import {
	canNext,
	currentIndex,
	queue,
	setSongYoutubeId,
} from "@/lib/music/views/stores/queue";
import { publish } from "@/lib/music/views/stores/remote";
import * as storage from "@/lib/shared/utils/storage";
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
// Bumped every time playback is asked to stop. A resume holds the element to
// actually starting for a few seconds, and a pause that lands inside that window
// must not leave a recovery behind that starts the track anyway.
let stops = 0;
// When the element was last paused. A stream that sat paused is the one that
// comes back silent, and a pause long enough to feel it is worth a loader while
// the resume proves itself.
let pausedAt = 0;
// Past this, a paused element's stream is more likely dead than not, and trying
// it plays the last buffered frames and nothing else: a blip of sound, then the
// rebuild. Measured on the phone, a pause of 45 s still resumed cleanly and one
// of a few minutes did not.
const TRUST_MS = 90_000;
// Set while a stream recovery reloads the shared element, so the transient
// load/pause it causes does not overwrite the stable UI. Cleared by the next
// `play` event or a timeout.
let quietReload = false;
let quietTimer: ReturnType<typeof setTimeout> | null = null;
// Stream recovery: re-resolve the dead stream and resume at the same position.
let recoveryTimer: ReturnType<typeof setTimeout> | null = null;
let recoveryInFlight = false;
let recoveryAttempt = 0;

let onTrackEndedCallback: (() => void) | null = null;
let endedElement: HTMLAudioElement | null = null;
// How many times this load was resumed from an end that came too early. A player
// restarts an ended element from the top, so an unchecked retry replays the track
// forever when the metadata duration is longer than the audio.
let retries = 0;
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

// Whether an event from this element belongs to the track the player thinks it
// holds. The silent autoplay prime reuses the shared element with its own clip,
// and reading that clip as the loaded song plays silence and skips the track.
function owns(a: HTMLAudioElement): boolean {
	return (
		audio === a &&
		endedElement !== a &&
		loadedSongId !== null &&
		AudioCache.holds(loadedSongId)
	);
}

function endPlayback(a: HTMLAudioElement) {
	if (!owns(a)) return;
	// A player that gives up mid-file (a range request it cut short) reports the
	// end early: the position is well short of the duration, so the track is not
	// over. Keep playing instead of ending it.
	const dur = knownDurationOr(a.duration);
	if (
		Number.isFinite(dur) &&
		dur > 0 &&
		a.currentTime < dur - 2 &&
		retries === 0
	) {
		retries++;
		// play() on an element that already reported `ended` seeks to the
		// beginning by specification, so the position it reached is put back: a
		// range the browser cut short continues instead of replaying the track.
		const resumeAt = a.currentTime;
		const playback = a.play();
		a.currentTime = resumeAt;
		void playback.catch(() => requestRecovery());
		return;
	}
	endedElement = a;
	stopEndWatch();
	clearBufferTimer();
	a.pause();
	progress.value = 0;
	isPlaying.value = false;
	media.update(currentSong.value, false);
	AudioCache.dropBuffers();
	onTrackEndedCallback?.();
}

let bufferTimer: ReturnType<typeof setTimeout> | null = null;

function clearBufferTimer() {
	if (bufferTimer !== null) {
		clearTimeout(bufferTimer);
		bufferTimer = null;
	}
}

// WebKit on macOS does not always fire `ended` for YouTube CDN streams, so end
// the track once it reaches its known duration or stalls right at the end.
function watchEnd(a: HTMLAudioElement) {
	stopEndWatch();
	let last = -1;
	let still = 0;
	endWatch = setInterval(() => {
		if (!owns(a) || a.ended) {
			stopEndWatch();
			return;
		}
		// A seek parks currentTime while the new range loads; that is buffering,
		// not a dead stream, so it must not trip the stall watchdog.
		if (a.paused || a.seeking) {
			last = -1;
			still = 0;
			return;
		}
		const elementDuration = a.duration;
		const time = a.currentTime;
		if (time === last) {
			still += 250;
		} else {
			still = 0;
			last = time;
			// The stream came back on its own: drop a reload a stall scheduled.
			if (recoveryTimer !== null) cancelRecovery();
		}
		if (shouldEnd(time, elementDuration, knownDuration, still)) {
			endPlayback(a);
			return;
		}
		// Do not treat ordinary buffering as a dead stream: a very long track
		// can take several seconds to fill its buffer after a seek, and firing
		// recovery there only re-resolves a URL that is still alive.
		if (stalled(still) && (still >= 8000 || !isLoading.value)) {
			still = 0;
			last = time;
			requestRecovery();
		}
	}, 250);
}

function endQuietReload() {
	quietReload = false;
	if (quietTimer !== null) {
		clearTimeout(quietTimer);
		quietTimer = null;
	}
}

function beginQuietReload() {
	quietReload = true;
	if (quietTimer !== null) clearTimeout(quietTimer);
	// If the reload never reaches `play`, stop hiding the UI after a short window
	// so a manual pause or an error still shows through.
	quietTimer = setTimeout(endQuietReload, 5000);
}

function cancelRecovery() {
	if (recoveryTimer !== null) {
		clearTimeout(recoveryTimer);
		recoveryTimer = null;
	}
	recoveryAttempt = 0;
}

// Position to restore after a rebuild: the live element when it still holds the
// song, otherwise the last position the UI reported.
function resumePosition(): number {
	if (audio && loadedSongId === currentSong.value?.id && !audio.error) {
		// The element can be sitting at zero while the UI still knows better: the
		// autoplay prime swaps a silent clip into the shared element on the first
		// gesture, and restoring the real stream lands a moment later. Taking the
		// later of the two keeps the position the user was at.
		return Math.max(audio.currentTime, progress.value);
	}
	return progress.value;
}

function scheduleRecovery(song: Song) {
	if (recoveryTimer !== null) return;
	const delay = retryDelay(recoveryAttempt++);
	recoveryTimer = setTimeout(() => {
		recoveryTimer = null;
		if (currentSong.value?.id !== song.id) {
			recoveryAttempt = 0;
			return;
		}
		void recover();
	}, delay);
}

// Entry point for the watchdogs: a no-op while a retry is already pending, so a
// stall that keeps firing every tick cannot bypass the backoff.
function requestRecovery() {
	if (recoveryTimer !== null || recoveryInFlight) {
		return;
	}
	void recover();
}

// A dead stream is not a broken song: re-resolve the URL (YouTube links expire),
// rebuild the element and resume at the same position. Retries forever with
// backoff, silently, and never advances the queue.
async function recover(): Promise<void> {
	if (recoveryInFlight) return;
	const song = currentSong.value;
	if (!song) return;

	// Offline: the resolve would only burn its timeout. Wait for the `online`
	// event or the next backoff tick.
	if (typeof navigator !== "undefined" && navigator.onLine === false) {
		scheduleRecovery(song);
		return;
	}

	recoveryInFlight = true;
	const at = resumePosition();
	try {
		AudioCache.remove(song.id);
		const loaded = await prepareSong(song, undefined, true);
		const el = audio;
		if (!loaded || !el) {
			scheduleRecovery(song);
			return;
		}
		seek(clampSeconds(at, knownDurationOr(el.duration)));
		const attempt = el.play();
		attempt.catch(() => {});
		// The same trap the resume path guards: a rebuilt stream that accepted
		// the play and never started is not a recovery, and counting it as one
		// stops the ladder with the session paused and nothing left to retry.
		if (!(await playing(el))) {
			throw new Error("recovery never started");
		}
		recoveryAttempt = 0;
		markFailed(song.id, false);
	} catch (err) {
		if (isNotAllowedError(err)) {
			blockedByPolicy = true;
			return;
		}
		scheduleRecovery(song);
	} finally {
		recoveryInFlight = false;
	}
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
	return parseStoredVolume(storage.get(VOLUME_STORAGE_KEY, null));
}

function getInitialMuted() {
	return parseStoredMuted(storage.get(MUTED_STORAGE_KEY, null));
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
	if (await waitForPlayable(el, 30000)) return;
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
		// Stop half a second short of the end: landing on the last frame reads as
		// a finished track, not as the position the user asked for.
		const eps = 0.5;
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

// Reflect a stopped load in the UI unless a quiet recovery owns the element and
// must keep the previous state visible while it rebuilds the stream.
function markStopped(): void {
	if (quietReload) return;
	isLoading.value = false;
	isPlaying.value = false;
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
	quiet = false,
): Promise<LoadStatus> {
	const token = ++loadToken;

	// A recovery reload hides its transient state; any other load is the normal
	// path and must clear a leftover quiet window from a failed recovery.
	if (quiet) beginQuietReload();
	else endQuietReload();

	stopEndWatch();
	const replaying = endedElement != null;
	endedElement = null;
	retries = 0;

	if (replaying) {
		AudioCache.releaseElement(song.id);
	}

	if (audio) {
		audio.pause();
		audio = null;
		loadedSongId = null;
		knownDuration = 0;
	}

	if (!quietReload) {
		progress.value = 0;
		audioDuration.value = 0;
		isLoading.value = true;
	}

	currentSong.value = song;
	media.update(song, isPlaying.value);

	let data: { url: string; youtube: string; duration: number };
	try {
		data = await AudioCache.get(song);
		knownDuration = data.duration > 0 ? data.duration : 0;
		if (data.youtube) setYoutube(song, data.youtube);
	} catch (err) {
		if (token !== loadToken) return "aborted";
		markStopped();
		logError(err, `resolve ${songLabel(song)}`);
		return "failed";
	}

	let instance: HTMLAudioElement;
	try {
		const preload = "auto";
		instance = await AudioCache.getAudioElement(song, preload);
	} catch (err) {
		if (token !== loadToken) return "aborted";
		markStopped();
		logError(err, `element ${songLabel(song)}`);
		return "failed";
	}
	if (token !== loadToken) return "aborted";

	audio = instance;
	loadedSongId = song.id;

	instance.volume = gain(volume.value);
	instance.muted = muted.value;
	if (!quietReload) instance.currentTime = 0;

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
				markStopped();
				logError(err, `buffer ${songLabel(song)} | ${mediaState(instance)}`);
				return "failed";
			}
			seek(startAt());
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
				markStopped();
				logError(err, `play ${songLabel(song)} | ${mediaState(instance)}`);
				return "failed";
			}
		}
	} else {
		if (!quietReload) isPlaying.value = false;
		try {
			await waitUntilBufferedEnough(instance);
		} catch (err) {
			markStopped();
			if (quietReload) {
				logError(err, `buffer ${songLabel(song)}`);
			} else {
				addError(err, `buffer ${songLabel(song)}`);
			}
			return "failed";
		}
		if (wantsStart) seek(startAt());
	}

	if (token !== loadToken) return "aborted";
	// A prepared (not playing) load ends here, so clear the loader. An autostart
	// leaves it to the playing/timeupdate events, which fire when sound is out.
	if (!quietReload && !autostart) isLoading.value = false;
	return "ok";
}

// Retry the element already loaded for this song, without rebuilding it or
// re-resolving the stream. The cheapest recovery step.
async function retryElement(song: Song): Promise<boolean> {
	if (!audio || !hasLoaded(song.id)) {
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

	cancelRecovery();
	blockedByPolicy = false;

	const success = (): PlayResult => {
		markFailed(song.id, false);
		return "ok";
	};

	if (superseded()) return "aborted";

	// A consensus round already prepared this exact stream (the host waits for the
	// room, the guest warms it): resume it instead of rebuilding the pipeline and
	// re-buffering the same track.
	if (startSeconds == null && hasLoaded(song.id) && audio) {
		const el = audio;
		try {
			await el.play();
			if (loadedSongId !== song.id) return "aborted";
			isPlaying.value = true;
			return success();
		} catch (err) {
			if (isNotAllowedError(err)) {
				blockedByPolicy = true;
				return blocked(song);
			}
			if (!isAbortError(err)) {
				logError(err, `resume ${songLabel(song)} | ${mediaState(el)}`);
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

// WebKit needs at least one user gesture before a media element may autoplay.
// Grab that grant on the shared player with a silent prime at the very first
// pointer/key, so a play that lands after a slow resolve is not rejected and
// the first click is enough.
let primedAutoplay = false;

function primeAutoplayOnGesture(): void {
	if (primedAutoplay || typeof window === "undefined") return;
	primedAutoplay = true;

	const prime = () => {
		window.removeEventListener("pointerdown", prime, true);
		window.removeEventListener("keydown", prime, true);
		// Already playing (a media key got there first): the grant is banked and
		// swapping the stream here would cut the track.
		if (audio && !audio.paused) return;
		// The prime swaps its silent clip into the shared element, so the stream
		// the paused track was warming goes back right after it: the grant stays
		// banked on the element, and the seek bar keeps the real duration and a
		// track to seek through.
		const song = loadedSongId === null ? null : currentSong.value;
		AudioCache.primeAutoplay();
		if (song) {
			// The prime leaves the element muted, and reloading the stream does not
			// go through the playback path that would restore it.
			void AudioCache.getAudioElement(song, "auto").then((el) => {
				el.muted = muted.value;
				el.volume = gain(volume.value);
			});
		}
	};

	window.addEventListener("pointerdown", prime, true);
	window.addEventListener("keydown", prime, true);
}

export async function prepareSong(
	song: Song,
	startSeconds?: number | (() => number),
	quiet = false,
): Promise<boolean> {
	return (await loadSongIntoPlayer(song, false, startSeconds, quiet)) === "ok";
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
	return audio !== null && !audio.error && owns(audio);
}

// True when the element currently holds this exact song, so callers do not
// mistake a still-loaded previous track for the one the queue points at.
export function hasLoaded(songId: string): boolean {
	return loadedSongId === songId && hasLoadedAudio();
}

// Start the track that is loaded, or rebuild the stream behind it when it no
// longer plays. This is what the media session's play control runs, so it must
// end in sound or in a recovery -- never in a silent "ok".
export async function resume(): Promise<boolean> {
	const el = audio;
	// A stream left paused for long enough is rebuilt instead of played: trying it
	// hands over the last frames it buffered -- a blip of sound -- and then needs
	// the rebuild anyway.
	const stale = pausedAt > 0 && Date.now() - pausedAt > TRUST_MS;
	if (el && hasLoadedAudio() && !stale) {
		// The stream may still be dead and the element accepts the play without a
		// sound, so this waits for actual audio -- but only for a moment: a resume
		// the user asked for cannot feel slow for a recovery it rarely needs.
		const stopsAt = stops;
		const forSong = currentSong.value?.id;
		// The promise is watched, not awaited: an element whose stream died can
		// leave play() pending for good, and this is the one control that must
		// never hang on it. `playing` answers on the speakers or on the clock.
		const attempt = el.play();
		let failure: unknown = null;
		attempt.catch((err: unknown) => {
			failure = err;
		});
		if (await playing(el, RESUME_MS)) return true;
		// A pause landed inside the window: this resume has nothing left to wait
		// for, and the loader belongs to whoever raised it.
		if (stops !== stopsAt) return false;
		// Another track landed: its own load owns the loader and the recovery, so
		// leave both alone -- turning the spinner off here is what made a song
		// tapped during a stale resume look loaded before it played.
		if (currentSong.value?.id !== forSong) return false;
		if (failure && isAbortError(failure)) return false;
		if (failure && isNotAllowedError(failure)) {
			blockedByPolicy = true;
			return false;
		}
		// Accepted and silent, or rejected outright: the URL this element resolved
		// has expired (a paused stream keeps it for hours), and nothing else would
		// retry it -- the stall watchdogs are stopped while paused. Rebuild the
		// stream instead of leaving the session paused for good.
		logError(
			failure ?? new Error("resume never started"),
			`resume ${songLabel(currentSong.value)} | ${mediaState(el)}`,
		);
		isLoading.value = true;
		requestRecovery();
		return false;
	}
	// The restored element may no longer hold the track (the silent autoplay
	// prime swaps it), so fall back to the queued current track instead of
	// leaving a fresh session's play control mute.
	const song = currentSong.value ?? queue.value[currentIndex.value] ?? null;
	if (!song) return false;
	if (stale && currentSong.value) {
		isLoading.value = true;
		requestRecovery();
		return false;
	}
	const result = await play(song);
	// A failed load here never bound events to a fresh element, so no `error`
	// would reach us: the ladder is the only thing that retries it. A blocked
	// policy is not a dead stream -- `blocked` already armed the gesture retry,
	// and a recovery here would fight it forever.
	if (result === "failed" && !blockedByPolicy) requestRecovery();
	return result === "ok";
}

export function seek(time: number) {
	if (!Number.isFinite(time)) return;
	// Clamp against the element's own duration: the resolved metadata rounds up,
	// and a target past the end parks the element at the very end, which the
	// player reports as a finished track.
	const target = audio ? clampSeconds(time, audio.duration) : Math.max(0, time);
	if (audio) {
		audio.currentTime = target;
		media.seek(target);
	}
	progress.value = target;
}

// Seek and wait for the element to actually land, so a caller never starts
// playback from the old position while the seek is still in flight. Resolves on
// the `seeked` event, right away when already there, and on a timeout so a
// stream that never fires it cannot stall the caller.
export function seekTo(time: number, timeoutMs = 5000): Promise<void> {
	const el = audio;
	if (!el || !Number.isFinite(time)) return Promise.resolve();

	const target = clampSeconds(time, el.duration);
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
	if (audio) audio.volume = gain(v);
	storage.set(VOLUME_STORAGE_KEY, String(v));
}

function setMuted(value: boolean) {
	muted.value = value;
	if (audio) audio.muted = muted.value;
	storage.set(MUTED_STORAGE_KEY, value ? "1" : "0");
}

export function pause(): boolean {
	stops += 1;
	pausedAt = Date.now();
	cancelRecovery();
	endQuietReload();
	const el = audio;
	if (el && hasLoadedAudio() && !el.paused) {
		el.pause();
		return true;
	}
	return false;
}

export async function togglePlayPause(): Promise<boolean> {
	const el = audio;
	if (el && hasLoadedAudio() && !el.paused) {
		// Through pause(): it ends the quiet reload and the recovery ladder, which
		// a bare el.pause() would leave running and the UI would stay "playing".
		pause();
		return false;
	}
	return await resume();
}

export function stopPlayer() {
	stops += 1;
	pausedAt = Date.now();
	cancelRecovery();
	endQuietReload();
	loadToken++;
	stopEndWatch();
	endedElement = null;
	clearBufferTimer();
	if (audio) {
		audio.pause();
		audio = null;
		loadedSongId = null;
		knownDuration = 0;
	}
	AudioCache.dropBuffers();
	currentSong.value = null;
	media.clear();
	progress.value = 0;
	audioDuration.value = 0;
	isPlaying.value = false;
	isLoading.value = false;
	pendingStart.value = false;
}

function bindEvents(a: HTMLAudioElement) {
	const syncDuration = () => {
		if (!owns(a)) return;
		const d = knownDurationOr(a.duration);
		audioDuration.value = Number.isFinite(d) && d > 0 ? d : 0;
	};

	a.onloadedmetadata = syncDuration;
	a.ondurationchange = syncDuration;

	a.onplay = () => {
		if (!owns(a)) return;
		endQuietReload();
		clearBufferTimer();
		endedElement = null;
		pausedAt = 0;
		isPlaying.value = true;
		pendingStart.value = false;
		media.update(currentSong.value, true);
		watchEnd(a);
	};

	// `play` fires once the browser accepts playback, which is still before the
	// stream reaches the speakers. Only `playing` means audio is actually out, so
	// the loader stays until then instead of vanishing into a silent gap.
	a.onplaying = () => {
		if (!owns(a)) return;
		clearBufferTimer();
		isLoading.value = false;
	};

	// Re-buffering, at the start or mid-track: the element is not producing sound,
	// so the loader belongs on. A quiet recovery reloads behind the scenes and
	// must show it just the same.
	a.onwaiting = () => {
		if (!owns(a)) return;
		// Delay the spinner so a seek that resolves in a few hundred ms does not
		// flash it. A real stall still shows it after the grace window.
		clearBufferTimer();
		bufferTimer = setTimeout(() => {
			if (owns(a)) isLoading.value = true;
		}, 200);
	};

	a.onpause = () => {
		if (!owns(a)) return;
		pausedAt = Date.now();
		// A reload pauses the element behind the scenes; keep the UI stable and let
		// the following `play` decide the state.
		if (quietReload) {
			stopEndWatch();
			return;
		}
		isPlaying.value = false;
		media.update(currentSong.value, false);
		stopEndWatch();
	};

	a.ontimeupdate = () => {
		if (!owns(a)) return;
		// While a seek is in flight the element still reports the old position;
		// keep the bar on the target the user picked and the loader on until it
		// lands, instead of clearing loading and snapping back.
		if (a.seeking) return;
		// Belt and braces: a frame advancing means the wait is over, even if a
		// WebKit quirk swallowed the `playing` event.
		clearBufferTimer();
		isLoading.value = false;
		if (!dragSeeking.value) progress.value = a.currentTime;
		media.position(a, knownDuration);
	};

	a.onended = () => {
		endPlayback(a);
	};

	a.onerror = () => {
		if (!owns(a)) return;
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
			// A mid-track error is usually the stream dying with the connection, not
			// a broken song: recover it instead of reporting and leaving it stopped.
			logError(new Error(`media error code ${code}`), detail);
			requestRecovery();
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

// The card offers what can actually be done, so its next button follows the
// queue and not only track changes: an edit that removes the last song has to
// take the button with it.
media.wireSkipNext(() => canNext.value);
effect(() => {
	canNext.value;
	media.refresh();
});

if (typeof window !== "undefined") {
	// Started while the page is on screen, which is the only way a timer here
	// keeps running once the app goes to the background.
	setInterval(tick, 1000);

	primeAutoplayOnGesture();
	// The network came back: retry immediately instead of waiting out the backoff.
	window.addEventListener("online", () => {
		cancelRecovery();
		requestRecovery();
	});
	window.addEventListener("storage", (e: StorageEvent) => {
		if (e.storageArea !== localStorage) return;
		if (e.key === VOLUME_STORAGE_KEY) {
			const v = parseStoredVolume(e.newValue);
			volume.value = v;
			if (audio) audio.volume = gain(v);
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

// A page the host reloaded carries on where the card was, and it asks rather
// than being told: only this side knows when it is ready, and a page that
// publishes the track it restored on the way in makes the card flicker.
function takeResume(): { seconds: number; play: boolean } | null {
	if (typeof window === "undefined") return null;
	const wails = (
		window as unknown as { wails?: { takeMediaResume?: () => string } }
	).wails;
	if (!wails || typeof wails.takeMediaResume !== "function") return null;
	try {
		const raw = wails.takeMediaResume();
		if (!raw) return null;
		const data = JSON.parse(raw) as { seconds?: unknown; play?: unknown };
		if (typeof data.seconds !== "number" || !Number.isFinite(data.seconds)) {
			return null;
		}
		return { seconds: Math.max(0, data.seconds), play: data.play === true };
	} catch {
		return null;
	}
}

function restorePlayback(): void {
	const song = queue.value[currentIndex.value] ?? null;
	if (!song) return;
	// Restore the track and load it while paused: the seek bar has its duration
	// before the first play, the loader reports the stream on its way, and the
	// first play does not have to resolve anything. The shared element is what
	// gets loaded, so a stale element must never pass for loaded (hasLoadedAudio).
	const pending = takeResume();
	currentSong.value = song;
	progress.value = pending?.seconds ?? 0;
	audioDuration.value = 0;
	isPlaying.value = false;
	// The card already shows the pause this page is recovering: pushing the
	// restored track at zero would make it jump to 00:00 and back.
	if (pending) isLoading.value = true;
	else media.update(song, false);
	void prepareSong(song, pending?.seconds).then((loaded) => {
		if (loaded && pending?.play) void resume();
	});
}

if (typeof window !== "undefined") {
	restorePlayback();
}
