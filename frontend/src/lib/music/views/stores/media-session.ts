import { ClearDiscordPresence, UpdateDiscordPresence } from "@bindings/app";
import { Events } from "@wailsio/runtime";
import type { Song } from "@/lib/music/model";
import { t } from "@/lib/shared/i18n";
import { logError } from "@/lib/shared/views/stores/errors";

type Controls = {
	play: () => void;
	pause: () => void;
	stop: () => void;
	seek: (seconds: number) => void;
	position: () => number;
};

type NativeBridge = {
	mediaUpdate?: (json: string) => void;
	mediaClear?: () => void;
	mediaTick?: () => void;
};

let next = () => {};
let previous = () => {};

let current: Song | null = null;
let playing = false;
let storedDuration = 0;
let storedPosition = 0;
let lastPush = 0;
let lastPositionState = 0;
let observedAt = 0;

function session() {
	if (typeof navigator === "undefined" || !("mediaSession" in navigator)) {
		return null;
	}
	return navigator.mediaSession;
}

function native(): NativeBridge | null {
	if (typeof window === "undefined") return null;
	const wails = (window as unknown as { wails?: NativeBridge }).wails;
	return wails && typeof wails.mediaUpdate === "function" ? wails : null;
}

function pushNative(force = false) {
	if (!current) return;
	const now = Date.now();
	if (!force && now - lastPush < 5000) return;
	lastPush = now;
	const payload = JSON.stringify({
		title: current.name,
		artist: (current.artists ?? []).map((artist) => artist.name).join(", "),
		album: current.album.title,
		artwork: current.album.covers?.[0] ?? "",
		playing,
		// The card offers what can actually be done: a next button with nothing
		// behind it is an interaction that answers with silence.
		next: canSkipNext(),
		loading: t("music.player.loading"),
		duration: storedDuration,
		position: storedPosition,
	});

	const bridge = native();
	if (bridge?.mediaUpdate) {
		bridge.mediaUpdate(payload);
	}
	void UpdateDiscordPresence(payload).catch(() => {});
}

export function refresh() {
	pushNative(true);
}

// What the app can skip to, asked at push time. The store wires it up, so this
// module keeps no dependency on the queue.
let canSkipNext: () => boolean = () => true;

export function wireSkipNext(ask: () => boolean) {
	canSkipNext = ask;
}

export function setup(controls: Controls) {
	const media = session();
	if (media) {
		const actions: Array<[MediaSessionAction, MediaSessionActionHandler]> = [
			["play", controls.play],
			["pause", controls.pause],
			["stop", controls.stop],
			["previoustrack", () => previous()],
			["nexttrack", () => next()],
			[
				"seekbackward",
				(details) =>
					controls.seek(controls.position() - (details.seekOffset ?? 10)),
			],
			[
				"seekforward",
				(details) =>
					controls.seek(controls.position() + (details.seekOffset ?? 10)),
			],
			[
				"seekto",
				(details) => {
					if (details.seekTime != null) controls.seek(details.seekTime);
				},
			],
		];

		for (const [action, handler] of actions) {
			try {
				media.setActionHandler(action, handler);
			} catch {}
		}
	}

	if (native()) {
		// The host had to reload the page to get an action through. It reaches
		// the app's own log so the phone can report it without a cable.
		Events.On("media:recover", (ev) => {
			logError(new Error("media page rescued"), JSON.stringify(ev.data ?? {}));
		});
		Events.On("media:action", (ev) => {
			// A zero-delay timer proves this page can still run its own tasks: a
			// hidden page Chromium stopped answers the injected handler and nothing
			// else, and the host reloads it as soon as this does not arrive.
			setTimeout(() => native()?.mediaTick?.(), 0);
			const data = ev.data as { action?: string; value?: number } | undefined;
			switch (data?.action) {
				case "play":
					controls.play();
					break;
				case "pause":
					controls.pause();
					break;
				case "next":
					next();
					break;
				case "previous":
					previous();
					break;
				case "stop":
					controls.stop();
					break;
				case "seek":
					if (typeof data.value === "number") controls.seek(data.value / 1000);
					break;
			}
		});
	}
}

export function tracks(handlers: { next: () => void; previous: () => void }) {
	next = handlers.next;
	previous = handlers.previous;
}

export function update(song: Song | null, isPlaying: boolean) {
	const media = session();
	if (media) {
		if (song) {
			media.metadata = new MediaMetadata({
				title: song.name,
				artist: (song.artists ?? []).map((artist) => artist.name).join(", "),
				album: song.album.title,
				artwork: song.album.covers?.[0]
					? [{ src: song.album.covers[0], sizes: "512x512" }]
					: [],
			});
		}
		media.playbackState = isPlaying ? "playing" : "paused";
	}

	const changed = song?.id !== current?.id;
	current = song;
	playing = isPlaying;
	if (song) {
		if (changed) {
			storedDuration = 0;
			storedPosition = 0;
		}
		lastPush = 0;
		pushNative(true);
	} else {
		native()?.mediaClear?.();
		void ClearDiscordPresence().catch(() => {});
	}
}

export function position(audio: HTMLAudioElement, durationOverride = 0) {
	const duration = durationOverride > 0 ? durationOverride : audio.duration;
	const now = Date.now();
	const elapsed = observedAt > 0 ? (now - observedAt) / 1000 : 0;
	const ahead = audio.currentTime - storedPosition;
	// The system extrapolates the position from the last push at the playback
	// rate, so playback needs no push. A position that the clock cannot explain
	// is a seek (or a duration that just arrived), and waiting for the throttle
	// leaves the notification bar on the old spot until it snaps.
	const jumped =
		ahead > elapsed + 1.5 ||
		ahead < -1.5 ||
		(Number.isFinite(duration) && Math.abs(duration - storedDuration) > 1);
	const media = session();
	if (
		media &&
		Number.isFinite(duration) &&
		duration > 0 &&
		Number.isFinite(audio.currentTime) &&
		(jumped || now - lastPositionState >= 1000)
	) {
		lastPositionState = now;
		try {
			media.setPositionState({
				duration,
				playbackRate: audio.playbackRate,
				position: Math.min(audio.currentTime, duration),
			});
		} catch {}
	}

	if (Number.isFinite(duration)) storedDuration = duration;
	if (Number.isFinite(audio.currentTime)) storedPosition = audio.currentTime;
	observedAt = now;
	pushNative(jumped);
}

export function seek(seconds: number) {
	if (!current || !Number.isFinite(seconds)) return;
	// A seek crosses more than the throttle window in one tick, and the element
	// only lands it a range request later: until then the system keeps
	// extrapolating from the old spot and the bar snaps back. Show it now.
	if (Math.abs(seconds - storedPosition) < 1) return;
	storedPosition = seconds;
	observedAt = Date.now();
	const media = session();
	if (media && storedDuration > 0) {
		try {
			media.setPositionState({
				duration: storedDuration,
				playbackRate: 1,
				position: Math.min(seconds, storedDuration),
			});
		} catch {}
	}
	pushNative(true);
}

export function clear() {
	const media = session();
	if (media) {
		media.metadata = null;
		media.playbackState = "none";
	}
	current = null;
	playing = false;
	native()?.mediaClear?.();
	void ClearDiscordPresence().catch(() => {});
}
