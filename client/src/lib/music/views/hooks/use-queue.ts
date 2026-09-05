import { useSignal } from "@preact/signals";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useRef } from "preact/hooks";
import {
	moveAfterCurrent,
	moveQueue,
	playAt,
	unqueue,
} from "@/lib/music/views/stores/player";
import { currentIndex, queue, showQueue } from "@/lib/music/views/stores/queue";

export function useQueuePanel() {
	return {
		open: showQueue.value,
		toggleOpen: () => {
			showQueue.value = !showQueue.value;
		},
		songs: queue.value,
		currentIndex: currentIndex.value,
		removeFromQueueAt: unqueue,
		moveQueueItem: moveQueue,
		moveJustBelowCurrentPlaying: moveAfterCurrent,
		playSongAtQueueIndex: playAt,
	};
}

const ROW_PX = 64;

export function useQueueView() {
	const panel = useQueuePanel();

	const parentRef = useRef<HTMLDivElement>(null);
	const dragFrom = useSignal<number | null>(null);
	const dragOver = useSignal<number | null>(null);

	const songs = panel.songs;
	const idx = panel.currentIndex;

	const indexFromClientY = (clientY: number) => {
		const el = parentRef.current;
		if (!el || songs.length === 0) return 0;

		const r = el.getBoundingClientRect();
		const rel = el.scrollTop + (clientY - r.top);

		return Math.max(0, Math.min(songs.length - 1, Math.floor(rel / ROW_PX)));
	};

	const clearDrag = () => {
		dragFrom.value = null;
		dragOver.value = null;
	};

	const rowVirtualizer = useVirtualizer({
		count: songs.length,
		getScrollElement: () => parentRef.current,
		estimateSize: () => ROW_PX,
		overscan: 10,
		getItemKey: (index) => `${songs[index]?.id ?? "x"}-${index}`,
	});

	useEffect(() => {
		if (!panel.open) return;

		const onKey = (e: KeyboardEvent) => {
			if (e.key === "Escape") {
				showQueue.value = false;
			}
		};

		window.addEventListener("keydown", onKey);
		return () => window.removeEventListener("keydown", onKey);
	}, [panel.open]);

	const close = () => {
		showQueue.value = false;
	};

	return {
		panel,
		songs,
		idx,

		parentRef,
		rowVirtualizer,

		dragFrom,
		dragOver,

		close,
		clearDrag,
		indexFromClientY,
	};
}
