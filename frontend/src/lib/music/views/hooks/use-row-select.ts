import { useRef } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { currentSong, isPlaying } from "@/lib/music/views/stores/audio";
import { toggleSong } from "@/lib/music/views/stores/player";
import {
	selectionActive,
	startSelection,
	toggleSelection,
} from "@/lib/music/views/stores/selection";

// A double click/tap must win over the single-click action, so the row waits
// this long before activating. The second click cancels the pending activation;
// a slower double click (past the window but still counted by the browser) has
// its already-started playback paused.
const DOUBLE_CLICK_MS = 250;

export function useRowSelect({
	song,
	onActivate,
	disabled = false,
}: {
	song: Song;
	onActivate: () => void;
	disabled?: boolean;
}) {
	const timer = useRef<number | null>(null);
	const startedPlayback = useRef(false);
	const suppressClick = useRef(false);
	const lastTap = useRef(0);

	const cancel = () => {
		if (timer.current == null) return;
		clearTimeout(timer.current);
		timer.current = null;
	};

	const enterSelection = () => {
		cancel();
		if (startedPlayback.current) {
			startedPlayback.current = false;
			if (currentSong.value?.id === song.id && isPlaying.value) {
				void toggleSong();
			}
		}
		if (!disabled) startSelection(song);
	};

	const onClick = (e: MouseEvent) => {
		if (suppressClick.current) {
			suppressClick.current = false;
			e.preventDefault();
			return;
		}
		if (selectionActive.value) {
			e.preventDefault();
			if (!disabled) toggleSelection(song);
			return;
		}
		if (disabled) return;
		// The browser increments detail on each click of a double click.
		if (e.detail >= 2) {
			e.preventDefault();
			enterSelection();
			return;
		}
		if (timer.current != null) return;
		startedPlayback.current = false;
		timer.current = window.setTimeout(() => {
			timer.current = null;
			startedPlayback.current = true;
			onActivate();
		}, DOUBLE_CLICK_MS);
	};

	// Touch has no reliable dblclick, so double taps are detected here.
	const onPointerDown = (e: PointerEvent) => {
		if (e.pointerType === "mouse") return;
		if (suppressClick.current || selectionActive.value || disabled) return;
		const now = Date.now();
		if (now - lastTap.current < DOUBLE_CLICK_MS) {
			lastTap.current = 0;
			suppressClick.current = true;
			enterSelection();
			return;
		}
		lastTap.current = now;
	};

	return { onClick, onPointerDown };
}
