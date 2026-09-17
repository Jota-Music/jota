import type { RefObject } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";

const CLOSE_PX = 96;
const FLING_PX_PER_MS = 0.5;
const DRAG_START_PX = 6;
const MOBILE_QUERY = "(max-width: 639px)";
const INTERACTIVE =
	"button, a, input, select, textarea, label, [role='slider'], [contenteditable='true']";

// Touch drag-to-dismiss for a bottom sheet. Ignores gestures that start on an
// interactive control or on a scroll container that is not at the top, so
// native scrolling keeps working.
export function useSheetDrag(
	panelRef: RefObject<HTMLDivElement>,
	backdropRef: RefObject<HTMLButtonElement>,
	close: () => void,
	mounted: boolean,
): boolean {
	const [dragging, setDragging] = useState(false);
	const closeRef = useRef(close);
	closeRef.current = close;
	const drag = useRef({
		active: false,
		possible: false,
		startY: 0,
		startTime: 0,
		offset: 0,
		scroll: null as HTMLElement | null,
	});

	useEffect(() => {
		if (!mounted) return;
		const panel = panelRef.current;
		const backdrop = backdropRef.current;
		if (!panel) return;
		const g = drag.current;

		const scrollableWithin = (node: HTMLElement) => {
			let el: HTMLElement | null = node;
			while (el && el !== panel) {
				const overflowY = getComputedStyle(el).overflowY;
				if (
					(overflowY === "auto" || overflowY === "scroll") &&
					el.scrollHeight > el.clientHeight + 1
				) {
					return el;
				}
				el = el.parentElement;
			}
			return null;
		};

		const finish = () => {
			if (!g.active) {
				g.possible = false;
				return;
			}
			g.active = false;
			g.possible = false;
			setDragging(false);
			if (backdrop) {
				backdrop.style.transition = "";
				backdrop.style.opacity = "";
			}
			const elapsed = performance.now() - g.startTime;
			const velocity = g.offset / Math.max(elapsed, 1);
			if (g.offset > CLOSE_PX || velocity > FLING_PX_PER_MS) {
				panel.style.transition = "translate 0.25s ease-out";
				panel.style.translate = "0 100%";
				closeRef.current();
			} else {
				panel.style.transition =
					"translate 0.3s cubic-bezier(0.32, 0.72, 0, 1)";
				panel.style.translate = "0 0";
			}
			g.offset = 0;
		};

		const onStart = (e: TouchEvent) => {
			if (!window.matchMedia(MOBILE_QUERY).matches) return;
			const touch = e.touches[0];
			if (!touch) return;
			if ((e.target as HTMLElement).closest(INTERACTIVE)) return;
			const scroll = scrollableWithin(e.target as HTMLElement);
			g.startY = touch.clientY;
			g.startTime = performance.now();
			g.offset = 0;
			g.scroll = scroll;
			g.active = false;
			g.possible = !scroll || scroll.scrollTop <= 0;
		};

		const onMove = (e: TouchEvent) => {
			if (!g.possible && !g.active) return;
			const touch = e.touches[0];
			if (!touch) return;
			const delta = touch.clientY - g.startY;
			if (!g.active) {
				if (delta < -DRAG_START_PX || (g.scroll && g.scroll.scrollTop > 0)) {
					g.possible = false;
					return;
				}
				if (delta < DRAG_START_PX) return;
				g.active = true;
				setDragging(true);
				panel.style.transition = "none";
				if (backdrop) backdrop.style.transition = "none";
			}
			e.preventDefault();
			g.offset = delta > 0 ? delta : 0;
			panel.style.translate = `0 ${g.offset}px`;
			if (backdrop) {
				const progress = Math.min(g.offset / CLOSE_PX, 1);
				backdrop.style.opacity = String(1 - progress * 0.85);
			}
		};

		panel.addEventListener("touchstart", onStart, { passive: true });
		panel.addEventListener("touchmove", onMove, { passive: false });
		panel.addEventListener("touchend", finish);
		panel.addEventListener("touchcancel", finish);
		return () => {
			panel.removeEventListener("touchstart", onStart);
			panel.removeEventListener("touchmove", onMove);
			panel.removeEventListener("touchend", finish);
			panel.removeEventListener("touchcancel", finish);
		};
	}, [mounted, panelRef, backdropRef]);

	return dragging;
}
