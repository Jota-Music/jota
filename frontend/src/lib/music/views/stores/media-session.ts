import type { Song } from "@/lib/music/model";

type Controls = {
	play: () => void;
	pause: () => void;
	stop: () => void;
	seek: (seconds: number) => void;
	position: () => number;
};

let next = () => {};
let previous = () => {};

function session() {
	if (typeof navigator === "undefined" || !("mediaSession" in navigator)) {
		return null;
	}
	return navigator.mediaSession;
}

export function setup(controls: Controls) {
	const media = session();
	if (!media) return;

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

export function tracks(handlers: { next: () => void; previous: () => void }) {
	next = handlers.next;
	previous = handlers.previous;
}

export function update(song: Song | null, playing: boolean) {
	const media = session();
	if (!media) return;

	if (song) {
		media.metadata = new MediaMetadata({
			title: song.name,
			artist: song.artists.map((artist) => artist.name).join(", "),
			album: song.album.title,
			artwork: song.album.covers?.[0]
				? [{ src: song.album.covers[0], sizes: "512x512" }]
				: [],
		});
	}
	media.playbackState = playing ? "playing" : "paused";
}

export function position(audio: HTMLAudioElement) {
	const media = session();
	if (
		!media ||
		!Number.isFinite(audio.duration) ||
		audio.duration <= 0 ||
		!Number.isFinite(audio.currentTime)
	)
		return;

	try {
		media.setPositionState({
			duration: audio.duration,
			playbackRate: audio.playbackRate,
			position: Math.min(audio.currentTime, audio.duration),
		});
	} catch {}
}

export function clear() {
	const media = session();
	if (!media) return;
	media.metadata = null;
	media.playbackState = "none";
}
