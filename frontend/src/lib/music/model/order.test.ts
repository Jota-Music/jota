import { expect, test } from "bun:test";
import { applyOrder, move } from "./order";

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
