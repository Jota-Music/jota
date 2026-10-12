import { expect, test } from "bun:test";
import { retryDelay, stalled } from "@/lib/music/app/recovery";

test("a brief buffering pause is not a stall", () => {
	expect(stalled(2000)).toBe(false);
});

test("a stream that stops advancing is a stall", () => {
	expect(stalled(4000)).toBe(true);
});

test("retry backoff doubles", () => {
	expect(retryDelay(0)).toBe(1000);
	expect(retryDelay(1)).toBe(2000);
	expect(retryDelay(2)).toBe(4000);
});

test("retry backoff is capped", () => {
	expect(retryDelay(10)).toBe(15000);
});
