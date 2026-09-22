import { expect, test } from "bun:test";
import { pinAfter, sameQueue } from "@/lib/music/app/queue";
import type { Song } from "@/lib/music/model";

const base = {
	url: "",
	duration: 0,
	share: { id: "", url: "" },
	album: { id: "", title: "", url: "", covers: [] },
	artists: [],
} satisfies Partial<Song>;

function song(id: string): Song {
	return { id, name: id, ...base };
}

test("pinAfter moves a song next to the current one", () => {
	const state = pinAfter([song("cur"), song("b"), song("c")], 0, 2);
	expect(state?.queue.map((s) => s.id)).toEqual(["cur", "c", "b"]);
	expect(state?.index).toBe(0);
});

test("pinAfter is a no-op right after the current one or out of range", () => {
	expect(pinAfter([song("cur"), song("b")], 0, 1)).toBeNull();
	expect(pinAfter([song("a")], 0, 3)).toBeNull();
});

test("sameQueue matches by membership, not order", () => {
	expect(
		sameQueue(
			[song("a"), song("b"), song("c")],
			[song("c"), song("a"), song("b")],
		),
	).toBe(true);
});

test("sameQueue respects duplicates", () => {
	expect(sameQueue([song("a"), song("a")], [song("a"), song("b")])).toBe(false);
	expect(sameQueue([song("a"), song("a")], [song("a"), song("a")])).toBe(true);
});

test("sameQueue rejects different lengths and ids", () => {
	expect(sameQueue([song("a")], [song("a"), song("b")])).toBe(false);
	expect(sameQueue([song("a")], [song("b")])).toBe(false);
});
