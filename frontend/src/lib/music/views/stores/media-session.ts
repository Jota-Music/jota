import { ClearDiscordPresence, UpdateDiscordPresence } from "@bindings/app";
import { Events } from "@wailsio/runtime";
import type { Song } from "@/lib/music/model";

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
};

let next = () => {};
let previous = () => {};

let current: Song | null = null;
let playing = false;
let storedDuration = 0;
let storedPosition = 0;
let lastPush = 0;

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
	if (!force && now - lastPush < 2000) return;
	lastPush = now;
	const payload = JSON.stringify({
		title: current.name,
		artist: (current.artists ?? []).map((artist) => artist.name).join(", "),
		album: current.album.title,
		artwork: current.album.covers?.[0] ?? "",
		playing,
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
		Events.On("media:action", (ev) => {
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
	const media = session();
	if (
		media &&
		Number.isFinite(duration) &&
		duration > 0 &&
		Number.isFinite(audio.currentTime)
	) {
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
	pushNative();
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
