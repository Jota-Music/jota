import { expect, test } from "bun:test";
import { offset, project, roundTrip } from "@/lib/sync/app/clock";

test("offset recovers a symmetric relay lead", () => {
	// The relay runs 1000ms ahead of the local clock; 20ms one way.
	const stamps = { t0: 1000, t1: 2020, t2: 2025, t3: 1045 };
	expect(offset(stamps)).toBeCloseTo(1000, 5);
	expect(roundTrip(stamps)).toBeCloseTo(40, 5);
});

test("offset absorbs an asymmetric route within the round trip", () => {
	// 30ms out, 10ms back: the offset is off by at most half the difference.
	const stamps = { t0: 0, t1: 1030, t2: 1030, t3: 40 };
	const err = Math.abs(offset(stamps) - 1000);
	expect(err).toBeLessThanOrEqual(10);
});

test("project advances only a playing position and clamps at zero", () => {
	expect(project(5000, 1000, 1500, true)).toBeCloseTo(5.5, 5);
	expect(project(5000, 1000, 1500, false)).toBeCloseTo(5, 5);
	expect(project(0, 5000, 1000, true)).toBe(0);
});

test("roundTrip never goes negative", () => {
	const stamps = { t0: 0, t1: 0, t2: 100, t3: 10 };
	expect(roundTrip(stamps)).toBe(0);
});
