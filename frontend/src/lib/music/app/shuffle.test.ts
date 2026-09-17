import { expect, test } from "bun:test";
import { permute, random, seed } from "@/lib/music/app/shuffle";

const items = ["a", "b", "c", "d", "e", "f", "g", "h"];

test("permute keeps every item", () => {
	const out = permute(items, random(1));
	expect([...out].sort()).toEqual([...items].sort());
});

test("the same seed yields the same order", () => {
	expect(permute(items, random(42))).toEqual(permute(items, random(42)));
});

test("different seeds yield different orders", () => {
	expect(permute(items, random(1))).not.toEqual(permute(items, random(2)));
});

test("permute handles empty and single-item lists", () => {
	expect(permute([], random(1))).toEqual([]);
	expect(permute(["only"], random(1))).toEqual(["only"]);
});

test("random is reproducible for a seed", () => {
	const a = random(7);
	const b = random(7);
	expect([a(), a(), a()]).toEqual([b(), b(), b()]);
});

test("seed returns an unsigned 32-bit integer", () => {
	const s = seed();
	expect(Number.isInteger(s)).toBe(true);
	expect(s).toBeGreaterThanOrEqual(0);
	expect(s).toBeLessThan(0x100000000);
});
