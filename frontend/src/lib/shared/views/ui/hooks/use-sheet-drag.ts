import type { RefObject } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";

const CLOSE_PX = 96;
const FLING_PX_PER_MS = 0.5;
const DRAG_START_PX = 6;
const INTERACTIVE =
	"button, a, input, select, textarea, label, [role='slider'], [contenteditable='true']";

function clientYOf(e: MouseEvent | TouchEvent): number {
	if ("touches" in e) {
		const touch = e.touches[0] ?? e.changedTouches[0];
		return touch ? touch.clientY : 0;
	}
	return e.clientY;
}

function isDraggableTarget(e: MouseEvent, panel: HTMLElement): boolean {
	const target = e.target as HTMLElement;
	if (target.closest(INTERACTIVE)) return false;
	let el: HTMLElement | null = target;
	while (el && el !== panel) {
		const overflowY = getComputedStyle(el).overflowY;
		if (
			(overflowY === "auto" || overflowY === "scroll") &&
			el.scrollHeight > el.clientHeight + 1
		) {
			return el.scrollTop <= 0;
		}
		el = el.parentElement;
	}
	return true;
}

// Drag-to-dismiss for a bottom sheet. Works with mouse and touch on all screen sizes.
// Ignores gestures that start on an interactive control or on a scroll container
// that is not at the top, so native scrolling keeps working.
export function useSheetDrag(
	panelRef: RefObject<HTMLDivElement>,
	backdropRef: RefObject<HTMLButtonElement>,
	close: () => void,
	mounted: boolean,
): boolean {
	const [dragging, setDragging] = useState(false);
	const [pressed, setPressed] = useState(false);
	const [canDrag, setCanDrag] = useState(false);
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
			setPressed(false);
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

		const onStart = (e: Event) => {
			const me = e as MouseEvent | TouchEvent;
			if ((me.target as HTMLElement).closest(INTERACTIVE)) return;
			const scroll = scrollableWithin(me.target as HTMLElement);
			if (!scroll || scroll.scrollTop <= 0) {
				setPressed(true);
			}
			g.startY = clientYOf(me);
			g.startTime = performance.now();
			g.offset = 0;
			g.scroll = scroll;
			g.active = false;
			g.possible = !scroll || scroll.scrollTop <= 0;
		};

		const onMove = (e: Event) => {
			if (!g.possible && !g.active) return;
			const me = e as MouseEvent | TouchEvent;
			const delta = clientYOf(me) - g.startY;
			if (!g.active) {
				if (delta < -DRAG_START_PX || (g.scroll && g.scroll.scrollTop > 0)) {
					g.possible = false;
					setPressed(false);
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

		const onMouseMoveHover = (e: Event) => {
			setCanDrag(isDraggableTarget(e as MouseEvent, panel));
		};

		const onMouseLeave = () => setCanDrag(false);

		const panelListeners: Array<
			[string, EventListener, boolean | AddEventListenerOptions]
		> = [
			["touchstart", onStart, { passive: true }],
			["mousedown", onStart, false],
			["mousemove", onMouseMoveHover, false],
			["mouseleave", onMouseLeave, false],
		];

		const windowListeners: Array<
			[string, EventListener, boolean | AddEventListenerOptions]
		> = [
			["touchmove", onMove, { passive: false }],
			["touchend", finish, false],
			["touchcancel", finish, false],
			["mousemove", onMove, false],
			["mouseup", finish, false],
		];

		for (const [type, handler, options] of panelListeners) {
			panel.addEventListener(type, handler, options);
		}
		for (const [type, handler, options] of windowListeners) {
			window.addEventListener(type, handler, options);
		}

		return () => {
			for (const [type, handler, options] of panelListeners) {
				panel.removeEventListener(type, handler, options);
			}
			for (const [type, handler, options] of windowListeners) {
				window.removeEventListener(type, handler, options);
			}
		};
	}, [mounted, panelRef, backdropRef]);

	useEffect(() => {
		if (dragging || pressed) {
			document.body.style.cursor = "grabbing";
		} else if (canDrag) {
			document.body.style.cursor = "grab";
		} else {
			document.body.style.cursor = "";
		}

		return () => {
			document.body.style.cursor = "";
		};
	}, [dragging, pressed, canDrag]);

	return dragging;
}
