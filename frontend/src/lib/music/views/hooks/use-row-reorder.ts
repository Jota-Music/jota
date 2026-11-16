import type { Signal } from "@preact/signals";
import type { RefObject } from "preact";
import { useRef } from "preact/hooks";
import { usePointerDrag } from "@/lib/shared/views/hooks/use-pointer-drag";

// ROW_PX is the height of one row: the window that renders the list and the hit
// test that picks the drop target must agree, or a drop lands on another row.
export const ROW_PX = 64;

type Options = {
	// from and over are the list's own drag signals, so two lists mounted at
	// once (the queue over a playlist) never highlight each other's rows.
	from: Signal<number | null>;
	over: Signal<number | null>;
	// row is the data attribute each row carries its index in.
	row: string;
	// base is the first index the window renders, for a list showing a slice.
	base?: number;
	enabled?: boolean;
	onMove: (from: number, to: number) => void;
};

// useRowReorder drags the rows of a fixed-height virtualized list: the row under
// the pointer is the drop target, and the drag carries the two indices.
export function useRowReorder(
	ref: RefObject<HTMLElement | null>,
	count: number,
	{ from, over, row, base = 0, enabled = true, onMove }: Options,
) {
	const source = useRef<number | null>(null);
	const attr = `data-${row}`;

	const indexAt = (clientY: number) => {
		const el = ref.current;
		if (!el || count === 0) return base;
		const rect = el.getBoundingClientRect();
		const rel = el.scrollTop + (clientY - rect.top);
		return base + Math.max(0, Math.min(count - 1, Math.floor(rel / ROW_PX)));
	};

	const { start, captureClick } = usePointerDrag({
		scroll: ref,
		touch: true,
		begin: () => {
			if (source.current == null) return;
			from.value = source.current;
			over.value = source.current;
		},
		move: (e) => {
			const next = indexAt(e.clientY);
			if (over.value !== next) over.value = next;
		},
		end: (dragged) => {
			const origin = from.value;
			const at = over.value;
			from.value = null;
			over.value = null;
			if (dragged && origin != null && at != null && origin !== at) {
				onMove(origin, at);
			}
		},
	});

	const pointerDown = (e: PointerEvent) => {
		if (!enabled) return;
		const target = e.target as Element;
		if (target.closest("button")) return;
		const el = target.closest(`[${attr}]`);
		if (!el) return;
		const index = Number(el.getAttribute(attr));
		if (!Number.isInteger(index)) return;
		if (!start(e)) return;
		source.current = index;
	};

	return { pointerDown, captureClick };
}
