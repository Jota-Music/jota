import { useEffect } from "preact/hooks";

import {
	currentSong,
	isLoading,
	isPlaying,
	pendingStart,
} from "@/lib/music/views/stores/audio";
import {
	nextSong,
	prevSong,
	toggleSong,
} from "@/lib/music/views/stores/player";
import { currentIndex, queue } from "@/lib/music/views/stores/queue";
import { binaryColor, dominantColor } from "@/lib/music/views/stores/theme";
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
			const binary = isDarkColor(r, g, b) ? "#FFFFFF" : "#000000";

			dominantColor.value = color;
			binaryColor.value = binary;
		})();

		return () => {
			cancelled = true;
		};
	}, [cover, songId]);

	if (!song) return null;

	const idx = currentIndex.value;
	const $queue = queue.value;

	return {
		song,
		cover,

		// state
		isLoading: isLoading.value || pendingStart.value,
		isPlaying: isPlaying.value,

		// controls
		toggleSong,
		nextSong,
		prevSong,

		// queue
		canPrev: idx > 0,
		canNext: idx < $queue.length - 1,
	};
}
