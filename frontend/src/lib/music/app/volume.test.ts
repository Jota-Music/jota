import { expect, test } from "bun:test";
import { gain } from "@/lib/music/app/volume";

test("maps the slider ends to silence and full scale", () => {
	expect(gain(0)).toBe(0);
	expect(gain(1)).toBe(1);
});

test("stays linear above the quiet floor", () => {
	expect(gain(0.3)).toBeCloseTo(0.2);
	expect(gain(0.65)).toBeCloseTo(0.6);
});

test("tapers toward silence below the quiet floor", () => {
	expect(gain(0.15)).toBeCloseTo(0.05);
	expect(gain(0.15)).toBeLessThan(0.15);
});

test("never leaves the 0..1 range audio.volume accepts", () => {
	for (let v = 0; v <= 1.0001; v += 0.01) {
		expect(gain(v)).toBeGreaterThanOrEqual(0);
		expect(gain(v)).toBeLessThanOrEqual(1);
	}
});
