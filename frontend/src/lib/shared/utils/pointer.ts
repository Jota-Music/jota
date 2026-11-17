// The frameless window resizes from the last pixels of its own sides. A gesture
// that claims that strip swallows the mouse events the resizer waits for, so
// the edge goes dead; the app's own gestures leave it alone.
const EDGE_PX = 8;

// Only a mouse feeds the resizer. A finger has none to feed, and on a phone the
// strip is where a thumb grabs the scrollbar: guarding it there eats half of a
// target that is already at the screen's own edge. A plain mouse event carries
// no pointerType, and that is the resizer's own input.
export function atWindowEdge(e: MouseEvent): boolean {
	const pointer = (e as PointerEvent).pointerType;
	if (pointer && pointer !== "mouse") return false;
	return e.clientX < EDGE_PX || e.clientX > window.innerWidth - EDGE_PX;
}

export function clientAxis(
	e: MouseEvent | TouchEvent,
	axis: "x" | "y",
): number {
	if ("touches" in e) {
		const touch = e.touches[0] ?? e.changedTouches[0];
		if (!touch) return 0;
		return axis === "x" ? touch.clientX : touch.clientY;
	}
	return axis === "x" ? e.clientX : e.clientY;
}
