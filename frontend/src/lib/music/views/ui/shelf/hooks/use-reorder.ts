import { signal } from "@preact/signals";
import type { RefObject } from "preact";
import { useRef } from "preact/hooks";
import { usePointerDrag } from "@/lib/shared/views/hooks/use-pointer-drag";

export const dragFrom = signal<string | null>(null);
export const dragOver = signal<string | null>(null);

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
		if (dragOver.value !== id) dragOver.value = id;
	};

	const dropItem = (id: string, e: DragEvent) => {
		const from = dragFrom.value;
		if (from == null) return;
		e.preventDefault();
		endDrag();
		if (from !== id) onReorder?.(from, id);
	};

	const source = useRef<string | null>(null);

	const { start, captureClick } = usePointerDrag({
		scroll: listRef,
		begin: () => {
			if (source.current == null) return;
			dragFrom.value = source.current;
			dragOver.value = source.current;
		},
		move: (e) => {
			const el = document.elementFromPoint(e.clientX, e.clientY);
			const id = el
				?.closest("[data-reorder-id]")
				?.getAttribute("data-reorder-id");
			if (id != null && dragOver.value !== id) dragOver.value = id;
		},
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
		draggable: reorderable && COARSE,
		onPointerDown: handlePointerDown(id),
		onDragStart: (e) => startDrag(id, e),
		onDragOver: (e) => overItem(id, e),
		onDrop: (e) => dropItem(id, e),
		onDragEnd: endDrag,
	});

	return { reorderable, dragProps, captureClick };
}
