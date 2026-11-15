import { expect, test } from "bun:test";
import { applyOrder, move, place } from "@/lib/shared/app/reorder";

const items = [{ id: "a" }, { id: "b" }, { id: "c" }, { id: "d" }];

test("applies the saved order", () => {
	expect(applyOrder(items, ["c", "a"]).map((i) => i.id)).toEqual([
		"c",
		"a",
		"b",
		"d",
	]);
});

test("keeps new items at the end in original order", () => {
	expect(applyOrder(items, ["d"]).map((i) => i.id)).toEqual([
		"d",
		"a",
		"b",
		"c",
	]);
});

test("ignores ids that no longer exist", () => {
	expect(applyOrder(items, ["gone", "b"]).map((i) => i.id)).toEqual([
		"b",
		"a",
		"c",
		"d",
	]);
});

test("no order leaves items untouched", () => {
	expect(applyOrder(items, [])).toEqual(items);
});

test("moves an item forward and backward", () => {
	expect(move(["a", "b", "c", "d"], "a", "c")).toEqual(["b", "c", "a", "d"]);
	expect(move(["a", "b", "c", "d"], "d", "b")).toEqual(["a", "d", "b", "c"]);
});

test("moving onto itself or unknown ids is a no-op", () => {
	expect(move(["a", "b"], "a", "a")).toEqual(["a", "b"]);
	expect(move(["a", "b"], "a", "x")).toEqual(["a", "b"]);
});

test("a drop takes the slot of the row it lands on", () => {
	expect(place(["a", "b", "c", "d"], 0, 2)).toEqual(["b", "c", "a", "d"]);
	expect(place(["a", "b", "c", "d"], 3, 1)).toEqual(["a", "d", "b", "c"]);
});

test("a drop lands where it was made, also on a reversed view", () => {
	const stored = ["a", "b", "c", "d", "e"];
	const view = [...stored].reverse();

	for (let from = 0; from < view.length; from++) {
		for (let to = 0; to < view.length; to++) {
			if (from === to) continue;
			const after = place(stored, from, to, true);
			expect([...after].reverse()[to]).toBe(view[from]);
		}
	}
});
