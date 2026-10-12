import { expect, test } from "bun:test";
import {
	delta,
	insertAt,
	isOwnList,
	pinAfter,
	sameQueue,
} from "@/lib/music/app/queue";
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

test("isOwnList matches the queue's source, ignoring the queue shape", () => {
	// A queue that gained a track (an enqueue) is still playing the list it
	// came from: the header play button must toggle instead of restarting.
	expect(
		isOwnList(
			"x",
			"x",
			[song("a"), song("b")],
			[song("a"), song("b"), song("c")],
		),
	).toBe(true);
	expect(
		isOwnList("y", "x", [song("a"), song("b")], [song("a"), song("b")]),
	).toBe(false);
});

test("isOwnList falls back to membership without a source", () => {
	expect(
		isOwnList(null, null, [song("a"), song("b")], [song("b"), song("a")]),
	).toBe(true);
	expect(isOwnList(null, null, [song("a")], [song("a"), song("b")])).toBe(
		false,
	);
});

test("insertAt lands the songs at the given position", () => {
	expect(
		insertAt([song("a"), song("b"), song("c")], [song("x")], 1).map(
			(s) => s.id,
		),
	).toEqual(["a", "x", "b", "c"]);
});

test("insertAt clamps a position past the end", () => {
	expect(
		insertAt([song("a"), song("b")], [song("x")], 9).map((s) => s.id),
	).toEqual(["a", "b", "x"]);
});

test("delta reports a contiguous insertion with its position", () => {
	expect(
		delta([song("a"), song("b")], [song("a"), song("x"), song("b")]),
	).toEqual({ action: "enqueue", songs: [song("x")], at: 1 });
	expect(delta([song("a")], [song("x"), song("a")])).toEqual({
		action: "enqueue",
		songs: [song("x")],
		at: 0,
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
