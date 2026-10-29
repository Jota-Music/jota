import { expect, test } from "bun:test";
import { within } from "@/lib/clipboard/app/within";

test("answers null for a provider that never settles", async () => {
	expect(await within(new Promise<string>(() => {}), 10)).toBeNull();
});

test("keeps an answer that arrives in time", async () => {
	expect(await within(Promise.resolve("text"), 1000)).toBe("text");
});

test("turns a rejection into null", async () => {
	expect(await within(Promise.reject(new Error("no")), 1000)).toBeNull();
});
