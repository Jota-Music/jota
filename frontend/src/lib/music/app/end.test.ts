import { expect, test } from "bun:test";
import { shouldEnd } from "@/lib/music/app/end";

test("ends on the known duration when WebKit reports Infinity", () => {
	expect(shouldEnd(214, Number.POSITIVE_INFINITY, 214, 0)).toBe(true);
});

test("keeps playing past the known duration ends the track", () => {
	expect(shouldEnd(240, Number.POSITIVE_INFINITY, 214, 0)).toBe(true);
});

test("does not end mid-track while the element advances", () => {
	expect(shouldEnd(100, Number.POSITIVE_INFINITY, 214, 0)).toBe(false);
});

test("ends on the element duration when it is reliable", () => {
	expect(shouldEnd(213.8, 214, 0, 0)).toBe(true);
	expect(shouldEnd(200, 214, 0, 0)).toBe(false);
});

test("ends on a stall right at the end without any duration", () => {
	expect(shouldEnd(213.5, Number.POSITIVE_INFINITY, 214, 1000)).toBe(true);
});

test("a mid-track stall does not end the song", () => {
	expect(shouldEnd(100, Number.POSITIVE_INFINITY, 214, 5000)).toBe(false);
});

test("no duration information never ends the song", () => {
	expect(shouldEnd(100, Number.POSITIVE_INFINITY, 0, 5000)).toBe(false);
});

test("time zero never ends the song", () => {
	expect(shouldEnd(0, 214, 214, 5000)).toBe(false);
});
