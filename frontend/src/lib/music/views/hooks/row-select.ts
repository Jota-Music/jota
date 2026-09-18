import type { Song } from "@/lib/music/model";
import {
	selectionActive,
	selectionAnchor,
	selectRange,
	setSelection,
	toggleSelection,
} from "@/lib/music/views/stores/selection";

export const coarse =
	typeof window !== "undefined" &&
	window.matchMedia("(pointer: coarse)").matches;

export function rowSelect({
	song,
	songs,
	index,
	onActivate,
	disabled = false,
}: {
	song: Song;
	songs: Song[];
	index: number;
	onActivate: () => void;
	disabled?: boolean;
}) {
	const onClick = (e: MouseEvent) => {
		if (disabled) return;

		if (coarse) {
			if (selectionActive.value) toggleSelection(song);
			else onActivate();
			return;
		}

		// The second click of a double click plays and collapses the selection.
		if (e.detail >= 2) {
			setSelection([song]);
			selectionAnchor.value = index;
			onActivate();
			return;
		}

		const anchor = selectionAnchor.value;
		if (e.shiftKey && selectionActive.value && anchor != null) {
			e.preventDefault();
			selectRange(songs, anchor, index);
			return;
		}

		if (e.ctrlKey || e.metaKey) {
			toggleSelection(song);
			selectionAnchor.value = index;
			return;
		}

		setSelection([song]);
		selectionAnchor.value = index;
	};

	return { onClick };
}
