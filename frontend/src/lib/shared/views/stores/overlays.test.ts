import { expect, test } from "bun:test";
import { dismiss, held, hold } from "@/lib/shared/views/stores/overlays";

test("dismiss reports nothing to close when no overlay is held", () => {
	expect(dismiss()).toBe(false);
});

test("dismiss closes the most recently held overlay", () => {
	let order: string[] = [];
	const releaseFirst = hold(() => order.push("first"));
	const releaseSecond = hold(() => order.push("second"));

	expect(dismiss()).toBe(true);
	expect(dismiss()).toBe(true);
	expect(order).toEqual(["second", "first"]);

	releaseSecond();
	releaseFirst();
});

test("hold returns a release that unregisters the overlay", () => {
	let closed = 0;
	const release = hold(() => closed++);

	release();
	expect(dismiss()).toBe(false);
	expect(closed).toBe(0);
});

test("held mirrors the depth so the back button can stay enabled", () => {
	expect(held.value).toBe(0);
	const releaseA = hold(() => {});
	const releaseB = hold(() => {});
	expect(held.value).toBe(2);

	dismiss();
	expect(held.value).toBe(1);
	releaseA();
	releaseB();
	expect(held.value).toBe(0);
});
