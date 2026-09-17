import { expect, test } from "bun:test";
import { pick } from "@/lib/music/app/playback";
import { clearQueued, insertQueued, pinAfter } from "@/lib/music/app/queue";
import type { QueueSong, Song } from "@/lib/music/model";

const base = {
	url: "",
	duration: 0,
	share: { id: "", url: "" },
	album: { id: "", title: "", url: "", covers: [] },
	artists: [],
} satisfies Partial<Song>;

function song(id: string, queued = false): QueueSong {
	return { id, name: id, ...base, queued };
}

test("insertQueued marks and places the song right after the current one", () => {
	const { queue, index } = insertQueued(
		[song("cur"), song("b"), song("c")],
		0,
		song("x"),
	);
	expect(queue.map((s) => s.id)).toEqual(["cur", "x", "b", "c"]);
	expect(queue[1].queued).toBe(true);
	expect(index).toBe(0);
});

test("insertQueued appends to an empty queue without marking it", () => {
	const { queue, index } = insertQueued([], -1, song("x"));
	expect(queue.map((s) => s.id)).toEqual(["x"]);
	expect(queue[0].queued).toBeFalsy();
	expect(index).toBe(0);
});

test("insertQueued appends when the current index is out of range", () => {
	const { queue } = insertQueued([song("a"), song("b")], 5, song("x"));
	expect(queue.map((s) => s.id)).toEqual(["a", "b", "x"]);
	expect(queue[2].queued).toBe(true);
});

test("an inserted song wins the next shuffle pick", () => {
	const { queue } = insertQueued(
		[song("cur"), song("b"), song("c")],
		0,
		song("x"),
	);
	expect(pick(queue, 0, "off", true, 1)).toBe(1);
});

test("pinAfter moves a song next to the current one and marks it queued", () => {
	const state = pinAfter([song("cur"), song("b"), song("c")], 0, 2);
	expect(state?.queue.map((s) => s.id)).toEqual(["cur", "c", "b"]);
	expect(state?.queue[1].queued).toBe(true);
	expect(state?.index).toBe(0);
});

test("pinAfter is a no-op right after the current one or out of range", () => {
	expect(pinAfter([song("cur"), song("b")], 0, 1)).toBeNull();
	expect(pinAfter([song("a")], 0, 3)).toBeNull();
});

test("clearQueued only unpins the played index", () => {
	const queue = [song("a", true), song("b", true)];
	const next = clearQueued(queue, 0);
	expect(next[0].queued).toBe(false);
	expect(next[1].queued).toBe(true);
});

test("clearQueued leaves an unqueued queue untouched", () => {
	const queue = [song("a")];
	expect(clearQueued(queue, 0)).toBe(queue);
});
