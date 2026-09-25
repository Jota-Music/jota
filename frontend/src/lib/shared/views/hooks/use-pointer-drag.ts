import type { RefObject } from "preact";
import { useEffect, useRef } from "preact/hooks";

const DRAG_START_PX = 4;
const HOLD_MS = 100;
const SCROLL_EDGE_PX = 48;
const SCROLL_MAX_STEP = 16;

type Options = {
	begin: () => void;
	move: (e: PointerEvent) => void;
	end: (dragged: boolean) => void;
	scroll?: RefObject<HTMLElement | null>;
};

// Mouse-only pointer drag. Native HTML5 drag lets the browser own the cursor,
// so the grabbing hand is only possible by driving the drag ourselves. Touch
// keeps using native drag, which already works on coarse pointers.
export function usePointerDrag({ begin, move, end, scroll }: Options) {
	const active = useRef(false);
	const moved = useRef(false);
	const dragged = useRef(false);
	const suppressClick = useRef(false);
	const origin = useRef({ x: 0, y: 0 });
	const timer = useRef<number | null>(null);
	const frame = useRef(0);
	const moveFrame = useRef(0);
	const last = useRef<PointerEvent | null>(null);
	const callbacks = useRef({ begin, move, end });
	callbacks.current = { begin, move, end };
	const scroller = useRef(scroll);
	scroller.current = scroll;

	const stopTimer = () => {
		if (timer.current == null) return;
		clearTimeout(timer.current);
		timer.current = null;
	};

	const stopScroll = () => {
		if (frame.current === 0) return;
		cancelAnimationFrame(frame.current);
		frame.current = 0;
	};

	// move may force layout (hit testing), so it runs at most once per frame
	// instead of on every pointermove the browser delivers.
	const runMove = () => {
		moveFrame.current = 0;
		const e = last.current;
		if (e) callbacks.current.move(e);
	};

	const scheduleMove = () => {
		if (moveFrame.current !== 0) return;
		moveFrame.current = requestAnimationFrame(runMove);
	};

	const stopMove = () => {
		if (moveFrame.current === 0) return;
		cancelAnimationFrame(moveFrame.current);
		moveFrame.current = 0;
	};

	// The last move must land before the drag ends, or a drop would resolve
	// against the previous frame's hover target.
	const flushMove = () => {
		if (moveFrame.current === 0) return;
		cancelAnimationFrame(moveFrame.current);
		runMove();
	};

	const tick = () => {
		frame.current = requestAnimationFrame(tick);
		const el = scroller.current?.current;
		const e = last.current;
		if (!el || !e) return;
		const rect = el.getBoundingClientRect();
		let delta = 0;
		if (e.clientY < rect.top + SCROLL_EDGE_PX) {
			delta = -Math.ceil(
				((rect.top + SCROLL_EDGE_PX - e.clientY) / SCROLL_EDGE_PX) *
					SCROLL_MAX_STEP,
			);
		} else if (e.clientY > rect.bottom - SCROLL_EDGE_PX) {
			delta = Math.ceil(
				((e.clientY - (rect.bottom - SCROLL_EDGE_PX)) / SCROLL_EDGE_PX) *
					SCROLL_MAX_STEP,
			);
		}
		delta = Math.max(-SCROLL_MAX_STEP, Math.min(SCROLL_MAX_STEP, delta));
		if (delta === 0) return;
		const before = el.scrollTop;
		el.scrollTop = before + delta;
		if (el.scrollTop !== before) scheduleMove();
	};

	const beginDrag = () => {
		if (!active.current || moved.current) return;
		stopTimer();
		moved.current = true;
		document.body.classList.add("pointer-dragging");
		callbacks.current.begin();
		stopScroll();
		frame.current = requestAnimationFrame(tick);
	};

	useEffect(() => {
		const onMove = (e: PointerEvent) => {
			if (!active.current) return;
			last.current = e;
			if (!moved.current) {
				const dx = e.clientX - origin.current.x;
				const dy = e.clientY - origin.current.y;
				if (dx * dx + dy * dy < DRAG_START_PX * DRAG_START_PX) return;
				beginDrag();
			}
			dragged.current = true;
			scheduleMove();
		};

		const onEnd = () => {
			stopTimer();
			stopScroll();
			flushMove();
			if (!active.current) return;
			active.current = false;
			last.current = null;
			document.body.classList.remove("pointer-dragging");
			const didDrag = dragged.current;
			if (didDrag) {
				suppressClick.current = true;
				setTimeout(() => {
					suppressClick.current = false;
				}, 0);
			}
			callbacks.current.end(didDrag);
		};

		window.addEventListener("pointermove", onMove);
		window.addEventListener("pointerup", onEnd);
		window.addEventListener("pointercancel", onEnd);
		return () => {
			stopTimer();
			stopScroll();
			stopMove();
			window.removeEventListener("pointermove", onMove);
			window.removeEventListener("pointerup", onEnd);
			window.removeEventListener("pointercancel", onEnd);
			document.body.classList.remove("pointer-dragging");
		};
	}, []);

	const start = (e: PointerEvent) => {
		if (e.pointerType !== "mouse" || e.button !== 0) return false;
		active.current = true;
		moved.current = false;
		dragged.current = false;
		origin.current = { x: e.clientX, y: e.clientY };
		(e.currentTarget as Element | null)?.setPointerCapture?.(e.pointerId);
		timer.current = window.setTimeout(beginDrag, HOLD_MS);
		return true;
	};

	const captureClick = (e: MouseEvent) => {
		if (!suppressClick.current) return;
		suppressClick.current = false;
		e.preventDefault();
		e.stopPropagation();
	};

	return { start, captureClick };
}
