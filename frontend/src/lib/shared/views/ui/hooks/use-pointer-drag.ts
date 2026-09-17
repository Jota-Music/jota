import { useEffect, useRef } from "preact/hooks";

const DRAG_START_PX = 4;
const HOLD_MS = 100;

type Options = {
	begin: () => void;
	move: (e: PointerEvent) => void;
	end: (dragged: boolean) => void;
};

// Mouse-only pointer drag. Native HTML5 drag lets the browser own the cursor,
// so the grabbing hand is only possible by driving the drag ourselves. Touch
// keeps using native drag, which already works on coarse pointers.
export function usePointerDrag({ begin, move, end }: Options) {
	const active = useRef(false);
	const moved = useRef(false);
	const dragged = useRef(false);
	const suppressClick = useRef(false);
	const origin = useRef({ x: 0, y: 0 });
	const timer = useRef<number | null>(null);
	const callbacks = useRef({ begin, move, end });
	callbacks.current = { begin, move, end };

	const stopTimer = () => {
		if (timer.current == null) return;
		clearTimeout(timer.current);
		timer.current = null;
	};

	const beginDrag = () => {
		if (!active.current || moved.current) return;
		stopTimer();
		moved.current = true;
		document.body.classList.add("pointer-dragging");
		callbacks.current.begin();
	};

	useEffect(() => {
		const onMove = (e: PointerEvent) => {
			if (!active.current) return;
			if (!moved.current) {
				const dx = e.clientX - origin.current.x;
				const dy = e.clientY - origin.current.y;
				if (dx * dx + dy * dy < DRAG_START_PX * DRAG_START_PX) return;
				beginDrag();
			}
			dragged.current = true;
			callbacks.current.move(e);
		};

		const onEnd = () => {
			stopTimer();
			if (!active.current) return;
			active.current = false;
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
