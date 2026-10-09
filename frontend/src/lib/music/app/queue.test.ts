import { expect, test } from "bun:test";
import { delta, insertAfter, pinAfter, sameQueue } from "@/lib/music/app/queue";
import type { Song } from "@/lib/music/model";

const base = {
	url: "",
	duration: 0,
	share: { id: "", url: "" },
	album: { id: "", title: "", url: "", covers: [] },
	artists: [],
} satisfies Partial<Song>;

function song(id: string, youtubeId?: string): Song {
	return { id, name: id, ...base, youtubeId };
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

test("insertAfter lands the songs after the named track", () => {
	const out = insertAfter([song("a"), song("b"), song("c")], [song("x")], "a");
	expect(out.queue.map((s) => s.id)).toEqual(["a", "x", "b", "c"]);
	expect(out.index).toBe(1);
});

test("insertAfter puts a null anchor at the head and a gone id at the end", () => {
	expect(
		insertAfter([song("a"), song("b")], [song("x")], null).queue.map(
			(s) => s.id,
		),
	).toEqual(["x", "a", "b"]);
	expect(
		insertAfter([song("a"), song("b")], [song("x")], "gone").queue.map(
			(s) => s.id,
		),
	).toEqual(["a", "b", "x"]);
});

test("delta reports a contiguous insertion with its anchor", () => {
	expect(
		delta([song("a"), song("b")], [song("a"), song("x"), song("b")]),
	).toEqual({ action: "enqueue", songs: [song("x")], afterId: "a" });
	expect(delta([song("a")], [song("x"), song("a")])).toEqual({
		action: "enqueue",
		songs: [song("x")],
		afterId: null,
	});
});

test("delta reports a single resolved YouTube id", () => {
	const op = delta([song("a"), song("b")], [song("a"), song("b", "vid")]);
	expect(op).toEqual({ action: "youtube", id: "b", youtubeId: "vid" });
});

test("delta refuses removals, reorders and unchanged queues", () => {
	expect(delta([song("a"), song("b")], [song("b"), song("a")])).toBeNull();
	expect(delta([song("a"), song("b")], [song("a")])).toBeNull();
	expect(delta([song("a")], [song("a")])).toBeNull();
	// A removal plus an insertion is not one incremental edit.
	expect(delta([song("a"), song("b")], [song("b"), song("x")])).toBeNull();
});
