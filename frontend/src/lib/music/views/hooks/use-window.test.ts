import { expect, test } from "bun:test";
import { band } from "@/lib/music/views/hooks/use-window";

const ROW = 64;
const OVERSCAN = 5;
const HEIGHT = 800;

test("a scroll inside the band keeps it identical", () => {
	const start = band(ROW * 20, HEIGHT, ROW, OVERSCAN);
	// Rows above the viewport stay rendered by the overscan, so moving a few
	// pixels down must not ask the list to render again.
	const moved = band(ROW * 20 + 20, HEIGHT, ROW, OVERSCAN);
	expect(moved).toEqual(start);
});

test("crossing a row moves the band", () => {
	const start = band(ROW * 20, HEIGHT, ROW, OVERSCAN);
	const moved = band(ROW * 21, HEIGHT, ROW, OVERSCAN);
	expect(moved.first).toBe(start.first + 1);
});

test("the band holds the visible rows plus the overscan, never negative", () => {
	expect(band(0, HEIGHT, ROW, OVERSCAN)).toEqual({
		first: 0,
		visible: Math.ceil(HEIGHT / ROW) + OVERSCAN * 2,
	});
});
