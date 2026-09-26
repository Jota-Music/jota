import type { RefObject } from "preact";
import { useEffect, useRef } from "preact/hooks";

const EDGE_PX = 48;
const MAX_STEP = 16;

// Edge autoscroll for any drag. The loop runs off a retained clientY, so a drag
// that stops moving keeps scrolling: native HTML5 drag — the only drag coarse
// pointers get — does not autoscroll on Android.
// ponytail: vertical only and a fixed px/frame cap, add an axis and an easing
// curve only if a drag ever needs to move something sideways.
export function useAutoscroll(
	scroll?: RefObject<HTMLElement | null>,
	onScroll?: () => void,
) {
	const clientY = useRef(0);
	const frame = useRef(0);
	const scroller = useRef(scroll);
	scroller.current = scroll;
	const scrolled = useRef(onScroll);
	scrolled.current = onScroll;

	const tick = () => {
		frame.current = requestAnimationFrame(tick);
		const el = scroller.current?.current;
		if (!el) return;
		const rect = el.getBoundingClientRect();
		let delta = 0;
		if (clientY.current < rect.top + EDGE_PX) {
			delta = -Math.ceil(
				((rect.top + EDGE_PX - clientY.current) / EDGE_PX) * MAX_STEP,
			);
		} else if (clientY.current > rect.bottom - EDGE_PX) {
			delta = Math.ceil(
				((clientY.current - (rect.bottom - EDGE_PX)) / EDGE_PX) * MAX_STEP,
			);
		}
		delta = Math.max(-MAX_STEP, Math.min(MAX_STEP, delta));
		if (delta === 0) return;
		const before = el.scrollTop;
		el.scrollTop = before + delta;
		// A list that cannot move further closes its own loop: dragend is not
		// guaranteed once the dragged row scrolls out of a virtualized window.
		if (el.scrollTop === before) {
			cancelAnimationFrame(frame.current);
			frame.current = 0;
			return;
		}
		scrolled.current?.();
	};

	const track = (y: number) => {
		clientY.current = y;
		if (frame.current !== 0) return;
		frame.current = requestAnimationFrame(tick);
	};

	const stop = () => {
		if (frame.current === 0) return;
		cancelAnimationFrame(frame.current);
		frame.current = 0;
	};

	useEffect(() => stop, []);

	return { track, stop };
}
