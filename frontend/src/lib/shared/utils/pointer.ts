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
