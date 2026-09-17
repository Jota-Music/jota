import { expect, test } from "bun:test";
import { pick } from "@/lib/music/app/playback";
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

test("next walks forward and stops at the end unless repeat all", () => {
	const q = [song("a"), song("b"), song("c")];
	expect(pick(q, 0, "off", 1)).toBe(1);
	expect(pick(q, 2, "off", 1)).toBeNull();
	expect(pick(q, 2, "all", 1)).toBe(0);
});

test("previous walks back and stops at the start unless repeat all", () => {
	const q = [song("a"), song("b"), song("c")];
	expect(pick(q, 2, "off", -1)).toBe(1);
	expect(pick(q, 0, "off", -1)).toBeNull();
	expect(pick(q, 0, "all", -1)).toBe(2);
});

test("repeat one stays on the same song", () => {
	const q = [song("a"), song("b")];
	expect(pick(q, 0, "one", 1)).toBe(0);
	expect(pick(q, 0, "one", -1)).toBe(0);
});

test("an empty queue has nowhere to go", () => {
	expect(pick([], 0, "off", 1)).toBeNull();
	expect(pick([], 0, "all", -1)).toBeNull();
});

test("a single-song queue stops unless repeat all", () => {
	const q = [song("only")];
	expect(pick(q, 0, "off", 1)).toBeNull();
	expect(pick(q, 0, "all", 1)).toBe(0);
});
