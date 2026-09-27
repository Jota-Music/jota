import type { RefObject } from "preact";
import { useEffect, useRef } from "preact/hooks";
import { useAutoscroll } from "@/lib/shared/views/hooks/use-autoscroll";

const DRAG_START_PX = 4;
const HOLD_MS = 250;

type Options = {
	begin: () => void;
	move: (e: PointerEvent) => void;
	end: (dragged: boolean) => void;
	scroll?: RefObject<HTMLElement | null>;
	// Touch reorders behind a long-press: a finger that moves straight away is
	// a scroll, and holding still is the row's own gesture, never a drag.
	touch?: boolean;
};

// Pointer drag. Native HTML5 drag lets the browser own the cursor, so the
// grabbing hand is only possible by driving the drag ourselves. The shelf
// keeps native drag behind its sort mode, where the long-press that opens an
// item menu is the same gesture the browser starts a drag with.
export function usePointerDrag({
	begin,
	move,
	end,
	scroll,
	touch = false,
}: Options) {
	const active = useRef(false);
	const armed = useRef(false);
	const hold = useRef(0);
	const moved = useRef(false);
	const dragged = useRef(false);
	const suppressClick = useRef(false);
	const origin = useRef({ x: 0, y: 0 });
	const moveFrame = useRef(0);
	const last = useRef<PointerEvent | null>(null);
	const capture = useRef<{ el: Element; id: number } | null>(null);
	const callbacks = useRef({ begin, move, end });
	callbacks.current = { begin, move, end };

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

	const autoscroll = useAutoscroll(scroll, scheduleMove);

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

	const beginDrag = () => {
		if (!active.current || moved.current) return;
		moved.current = true;
		// Capture only now, never on pointerdown. Capture retargets mousedown and
		// mouseup to this element, and click/dblclick target the nearest common
		// ancestor of those two: the scroller. An event dispatched at an ancestor
		// never reaches the rows, so their click handlers would be dead.
		capture.current?.el.setPointerCapture(capture.current.id);
		document.body.classList.add("pointer-dragging");
		callbacks.current.begin();
	};

	const clearHold = () => {
		if (hold.current === 0) return;
		clearTimeout(hold.current);
		hold.current = 0;
	};

	useEffect(() => {
		const onMove = (e: PointerEvent) => {
			if (!active.current) return;
			// Moving before the hold completes means the finger is scrolling, not
			// dragging.
			clearHold();
			last.current = e;
			if (!moved.current) {
				const dx = e.clientX - origin.current.x;
				const dy = e.clientY - origin.current.y;
				if (dx * dx + dy * dy < DRAG_START_PX * DRAG_START_PX) return;
				beginDrag();
			}
			dragged.current = true;
			scheduleMove();
			autoscroll.track(e.clientY);
		};

		const onEnd = () => {
			clearHold();
			autoscroll.stop();
			flushMove();
			if (!active.current) return;
			active.current = false;
			armed.current = false;
			last.current = null;
			capture.current = null;
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

		// A held finger scrolls the pane on its own, so the drag owns the touch
		// only once the long-press has armed it.
		const onTouchMove = (e: TouchEvent) => {
			if (armed.current) e.preventDefault();
		};

		window.addEventListener("pointermove", onMove);
		window.addEventListener("pointerup", onEnd);
		window.addEventListener("pointercancel", onEnd);
		window.addEventListener("touchmove", onTouchMove, { passive: false });
		return () => {
			clearHold();
			stopMove();
			window.removeEventListener("pointermove", onMove);
			window.removeEventListener("pointerup", onEnd);
			window.removeEventListener("pointercancel", onEnd);
			window.removeEventListener("touchmove", onTouchMove);
			document.body.classList.remove("pointer-dragging");
		};
	}, []);

	const start = (e: PointerEvent) => {
		if (e.pointerType === "mouse") {
			if (e.button !== 0) return false;
		} else if (!touch) {
			return false;
		}
		active.current = true;
		armed.current = false;
		moved.current = false;
		dragged.current = false;
		origin.current = { x: e.clientX, y: e.clientY };
		const el = e.currentTarget as Element | null;
		capture.current = el?.setPointerCapture ? { el, id: e.pointerId } : null;
		if (e.pointerType !== "mouse") {
			hold.current = window.setTimeout(() => {
				hold.current = 0;
				armed.current = true;
			}, HOLD_MS);
		}
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
