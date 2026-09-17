import { expect, test } from "bun:test";
import { nextQueued, pick } from "@/lib/music/app/playback";
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

test("shuffle plays the queued song before the pool", () => {
	const q = [song("cur"), song("b"), song("c"), song("next", true), song("e")];
	expect(pick(q, 0, "off", true, 1)).toBe(3);
});

test("shuffle wraps to a queued song behind the current one", () => {
	const q = [song("queued", true), song("b"), song("cur")];
	expect(pick(q, 2, "off", true, 1)).toBe(0);
});

test("shuffle never picks the current song", () => {
	const q = [song("a"), song("b"), song("c")];
	for (let k = 0; k < 50; k++) {
		expect(pick(q, 1, "off", true, 1)).not.toBe(1);
	}
});

test("repeat one ignores queued and shuffle", () => {
	const q = [song("a"), song("queued", true)];
	expect(pick(q, 0, "one", true, 1)).toBe(0);
});

test("shuffle previous stays random, not queued", () => {
	const q = [song("a"), song("queued", true), song("cur")];
	let sawOther = false;
	for (let k = 0; k < 200 && !sawOther; k++) {
		if (pick(q, 2, "off", true, -1) === 0) sawOther = true;
	}
	expect(sawOther).toBe(true);
});

test("no shuffle keeps linear order", () => {
	const q = [song("a"), song("b"), song("c")];
	expect(pick(q, 0, "off", false, 1)).toBe(1);
	expect(pick(q, 2, "off", false, 1)).toBeNull();
	expect(pick(q, 2, "all", false, 1)).toBe(0);
});

test("previous keeps linear order and wraps only on repeat all", () => {
	const q = [song("a"), song("b"), song("c")];
	expect(pick(q, 2, "off", false, -1)).toBe(1);
	expect(pick(q, 0, "off", false, -1)).toBeNull();
	expect(pick(q, 0, "all", false, -1)).toBe(2);
});

test("shuffle picks the nearest queued song after the current one", () => {
	const q = [song("cur"), song("near", true), song("far", true), song("z")];
	expect(pick(q, 0, "off", true, 1)).toBe(1);
});

test("an empty queue has nowhere to go", () => {
	expect(pick([], 0, "off", true, 1)).toBeNull();
	expect(pick([], 0, "off", false, 1)).toBeNull();
});

test("a single-song queue stops unless repeat all", () => {
	const q = [song("only")];
	expect(pick(q, 0, "off", true, 1)).toBeNull();
	expect(pick(q, 0, "off", false, 1)).toBeNull();
	expect(pick(q, 0, "all", false, 1)).toBe(0);
});

test("nextQueued is null without a queue or queued songs", () => {
	expect(nextQueued([], 0)).toBeNull();
	expect(nextQueued([song("a"), song("b")], 0)).toBeNull();
});

test("nextQueued skips the current index", () => {
	const q = [song("a"), song("cur", true), song("b", true)];
	expect(nextQueued(q, 1)).toBe(2);
	expect(nextQueued([song("a")], 0)).toBeNull();
});
