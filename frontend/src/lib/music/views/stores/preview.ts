import { effect, signal } from "@preact/signals";
import { gain } from "@/lib/music/app/volume";
import type { Song } from "@/lib/music/model";
import {
	isPlaying,
	muted,
	pause as pauseMain,
	resume as resumeMain,
	volume,
} from "@/lib/music/views/stores/audio";
import { AudioCache, SILENT_WAV } from "@/lib/music/views/stores/cache";
import { t } from "@/lib/shared/i18n";
import { addError } from "@/lib/shared/views/stores/errors";

export const previewing = signal<string | null>(null);
export const previewPending = signal<string | null>(null);
export const previewPlaying = signal(false);
export const previewProgress = signal(0);
export const previewDuration = signal(0);
export const previewSeeking = signal(false);

let player: HTMLAudioElement | null = null;
let token = 0;
let resumeMainAfterPreview = false;

function applyVolume(audio: HTMLAudioElement) {
	audio.volume = gain(volume.value);
	audio.muted = muted.value;
}

effect(() => {
	if (player) applyVolume(player);
});

function element(): HTMLAudioElement {
	if (player) return player;

	const audio = new Audio();
	audio.preload = "auto";
	audio.addEventListener("ended", stopPreview);
	audio.addEventListener("error", stopPreview);
	audio.addEventListener("timeupdate", () => {
		if (!previewSeeking.value) previewProgress.value = audio.currentTime;
	});
	audio.addEventListener("loadedmetadata", () => {
		const duration = audio.duration;
		if (Number.isFinite(duration) && duration > 0) {
			previewDuration.value = duration;
		}
	});
	player = audio;
	return audio;
}

// WebKit grants autoplay per element, so the silent clip must play inside the
// click that starts the preview. The element keeps that grant for later sources.
function prime(audio: HTMLAudioElement) {
	audio.muted = true;
	audio.src = SILENT_WAV;
	audio.load();
	void audio.play().then(
		() => {
			audio.pause();
			audio.muted = false;
		},
		() => {
			audio.muted = false;
		},
	);
}

function clearAudio() {
	token++;
	player?.pause();
	player?.removeAttribute("src");
	player?.load();
	previewing.value = null;
	previewPending.value = null;
	previewPlaying.value = false;
	previewProgress.value = 0;
	previewDuration.value = 0;
	previewSeeking.value = false;
}

export function stopPreview() {
	clearAudio();
	if (!resumeMainAfterPreview) return;
	resumeMainAfterPreview = false;
	void resumeMain();
}

// A deliberate player action takes ownership from the preview. Stop the
// separate element without restoring the track that the preview paused.
export function interruptPreview() {
	clearAudio();
	resumeMainAfterPreview = false;
}

async function playAudio() {
	const audio = player;
	if (!audio) return;

	try {
		await audio.play();
		previewPlaying.value = true;
	} catch (error) {
		stopPreview();
		addError(error, t("music.youtubeId.previewFailed"));
	}
}

export function togglePreview() {
	const audio = player;
	if (!audio || !previewing.value) return;

	if (previewPending.value) {
		stopPreview();
		return;
	}

	if (audio.paused) {
		void playAudio();
	} else {
		audio.pause();
		previewPlaying.value = false;
	}
}

export function seekPreview(seconds: number) {
	const audio = player;
	if (!audio) return;

	const max = previewDuration.value;
	const target =
		Number.isFinite(max) && max > 0 ? Math.min(seconds, max) : seconds;
	audio.currentTime = Math.max(0, target);
	previewProgress.value = audio.currentTime;
}

export async function preview(song: Song) {
	if (previewing.value === song.id) {
		if (!previewPending.value && player?.paused) await playAudio();
		return;
	}

	clearAudio();
	if (!resumeMainAfterPreview && isPlaying.value && pauseMain()) {
		resumeMainAfterPreview = true;
	}

	const audio = element();
	prime(audio);

	const current = ++token;
	previewing.value = song.id;
	previewPending.value = song.id;

	try {
		const cached = await AudioCache.get(song);
		if (current !== token) return;

		audio.src = cached.url;
		audio.load();
		applyVolume(audio);
		previewDuration.value = cached.duration > 0 ? cached.duration : 0;
		await audio.play();
		if (current !== token) return;

		previewPending.value = null;
		previewPlaying.value = true;
	} catch (error) {
		if (current !== token) return;
		stopPreview();
		addError(error, t("music.youtubeId.previewFailed"));
	}
}
