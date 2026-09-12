import { useEffect } from "preact/hooks";

import {
	audioDuration,
	currentSong,
	isLoading,
	isPlaying,
	progress,
} from "@/lib/music/views/stores/audio";
import { AudioCache } from "@/lib/music/views/stores/cache";
import {
	nextSong,
	prevSong,
	seekFromLocalControl,
	toggleSong,
} from "@/lib/music/views/stores/player";
import { currentIndex, queue } from "@/lib/music/views/stores/queue";
import {
	getDominantColorFromImage,
	isDarkColor,
	normalizeColor,
} from "@/lib/shared/utils/color";

export function usePlayer() {
	const song = currentSong.value;
	const songId = song?.id;
	const cover = song?.album.covers?.[0] ?? "";

	useEffect(() => {
		if (!songId || !cover) return;

		let cancelled = false;

		(async () => {
			let rgb: [number, number, number] | null = null;
			rgb = await getDominantColorFromImage(cover);

			if (cancelled || !rgb) return;

			const [r, g, b] = normalizeColor(rgb[0], rgb[1], rgb[2]);
			const color = `rgb(${r}, ${g}, ${b})`;

			document.documentElement.style.setProperty("--dominant-color", color);
			document.documentElement.style.setProperty(
				"--binary-color",
				isDarkColor(r, g, b) ? "#FFFFFF" : "#000000",
			);
		})();

		return () => {
			cancelled = true;
		};
	}, [cover, songId]);

	useEffect(() => {
		if (!song) return;

		let cancelled = false;

		(async () => {
			try {
				const cached = await AudioCache.get(song);
				if (!cancelled && cached.youtube) {
					const current = currentSong.value;
					if (current && current.id === song.id) {
						currentSong.value = { ...current, youtubeId: cached.youtube };
					}
				}
			} catch (error) {
				console.error("Error loading YouTube ID:", error);
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [songId]);

	if (!song) return null;

	const idx = currentIndex.value;
	const $queue = queue.value;

	return {
		song,
		cover,

		// state
		isLoading: isLoading.value,
		isPlaying: isPlaying.value,
		progress: progress.value,
		duration: audioDuration.value,

		// controls
		seek: seekFromLocalControl,
		toggleSong,
		nextSong,
		prevSong,

		// queue
		canPrev: idx > 0,
		canNext: idx < $queue.length - 1,
	};
}
