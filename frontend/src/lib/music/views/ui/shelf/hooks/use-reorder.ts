import { signal } from "@preact/signals";
import type { RefObject } from "preact";
import { useRef } from "preact/hooks";
import { useAutoscroll } from "@/lib/shared/views/hooks/use-autoscroll";
import { usePointerDrag } from "@/lib/shared/views/hooks/use-pointer-drag";

export const dragFrom = signal<string | null>(null);
export const dragOver = signal<string | null>(null);

// Reorder mode is a touch-only mode: on coarse pointers the long-press that
// opens the item menu is the same gesture the browser uses to start a drag.
export const sortMode = signal(false);

export function toggleSortMode() {
	sortMode.value = !sortMode.value;
}

export const COARSE = window.matchMedia("(pointer: coarse)").matches;

export interface DragProps {
	"data-reorder-id": string;
	draggable: boolean;
	onPointerDown: (e: PointerEvent) => void;
	onDragStart: (e: DragEvent) => void;
	onDragOver: (e: DragEvent) => void;
	onDrop: (e: DragEvent) => void;
	onDragEnd: (e: DragEvent) => void;
}

type Options = {
	count: number;
	listRef: RefObject<HTMLDivElement | null>;
	onReorder?: (fromId: string, toId: string) => void;
};

export function useReorder({ count, listRef, onReorder }: Options) {
	const reorderable = !!onReorder && count > 1;
	// Desktop drags from anywhere, always. Touch needs the toggle, and while
	// sorting the item menu steps aside for the drag.
	const canSort = reorderable && COARSE;
	const sorting = canSort && sortMode.value;

	const endDrag = () => {
		dragFrom.value = null;
		dragOver.value = null;
	};

	const startDrag = (id: string, e: DragEvent) => {
		if ((e.target as Element | null)?.closest("button")) {
			e.preventDefault();
			return;
		}
		dragFrom.value = id;
		const dt = e.dataTransfer;
		if (!dt) return;
		try {
			dt.setData("text/plain", id);
			dt.effectAllowed = "move";
		} catch {
			/* noop */
		}
	};

	const source = useRef<string | null>(null);
	const pointer = useRef({ x: 0, y: 0 });

	const hover = (x: number, y: number) => {
		const el = document.elementFromPoint(x, y);
		const id = el
			?.closest("[data-reorder-id]")
			?.getAttribute("data-reorder-id");
		if (id != null && dragOver.value !== id) dragOver.value = id;
	};

	const autoscroll = useAutoscroll(listRef, () =>
		hover(pointer.current.x, pointer.current.y),
	);

	const endItemDrag = () => {
		autoscroll.stop();
		endDrag();
	};

	const overItem = (id: string, e: DragEvent) => {
		if (dragFrom.value == null) return;
		e.preventDefault();
		const dt = e.dataTransfer;
		if (dt) {
			try {
				dt.dropEffect = "move";
			} catch {
				/* noop */
			}
		}
		pointer.current = { x: e.clientX, y: e.clientY };
		autoscroll.track(e.clientY);
		if (dragOver.value !== id) dragOver.value = id;
	};

	const dropItem = (id: string, e: DragEvent) => {
		const from = dragFrom.value;
		if (from == null) return;
		e.preventDefault();
		autoscroll.stop();
		endDrag();
		if (from !== id) onReorder?.(from, id);
	};

	const { start, captureClick } = usePointerDrag({
		scroll: listRef,
		begin: () => {
			if (source.current == null) return;
			dragFrom.value = source.current;
			dragOver.value = source.current;
		},
		move: (e) => hover(e.clientX, e.clientY),
		end: (dragged) => {
			const from = dragFrom.value;
			const to = dragOver.value;
			endDrag();
			if (dragged && from != null && to != null && from !== to) {
				onReorder?.(from, to);
			}
		},
	});

	const handlePointerDown = (id: string) => (e: PointerEvent) => {
		if (!reorderable) return;
		if ((e.target as Element).closest("button")) return;
		if (!start(e)) return;
		source.current = id;
	};

	const dragProps = (id: string): DragProps => ({
		"data-reorder-id": id,
		draggable: sorting,
		onPointerDown: handlePointerDown(id),
		onDragStart: (e) => startDrag(id, e),
		onDragOver: (e) => overItem(id, e),
		onDrop: (e) => dropItem(id, e),
		onDragEnd: endItemDrag,
	});

	return { reorderable, canSort, sorting, dragProps, captureClick };
}
