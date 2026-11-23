import { beforeEach, expect, test } from "bun:test";
import { permute, random } from "@/lib/music/app/shuffle";
import type { Song } from "@/lib/music/model";
import {
	applyShuffle,
	canNext,
	clampIndex,
	clearHeard,
	currentIndex,
	cycleRepeat,
	queue,
	remember,
	repeat,
	replaceQueue,
	setRepeat,
	setSongYoutubeId,
	startTrail,
	unwind,
} from "@/lib/music/views/stores/queue";

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

beforeEach(() => {
	queue.value = [];
	currentIndex.value = 0;
	clearHeard();
});

test("setSongYoutubeId fills the resolved id", () => {
	queue.value = [song("a")];
	setSongYoutubeId("a", "vid");
	expect(queue.value[0].youtubeId).toBe("vid");
});

test("setSongYoutubeId ignores unknown songs and no-op updates", () => {
	queue.value = [song("a", "vid")];
	setSongYoutubeId("missing", "x");
	setSongYoutubeId("a", "vid");
	expect(queue.value).toHaveLength(1);
	expect(queue.value[0].youtubeId).toBe("vid");
});

test("applyShuffle reorders the queue for the announced seed", () => {
	const songs = [song("a"), song("b"), song("c"), song("d")];
	queue.value = songs.slice();
	currentIndex.value = 0;

	applyShuffle(true, 123);

	expect(queue.value.map((s) => s.id)).toEqual(
		permute(songs, random(123)).map((s) => s.id),
	);
	expect(queue.value[currentIndex.value].id).toBe("a");
});

test("applyShuffle off keeps the current order", () => {
	queue.value = [song("b"), song("a")];
	currentIndex.value = 0;

	applyShuffle(false);

	expect(queue.value.map((s) => s.id)).toEqual(["b", "a"]);
});

test("unwind goes back to the track that played, not the shuffled neighbour", () => {
	// "a" played, then the user clicked "y": shuffle left "z" between them
	// without ever playing it.
	const songs = [song("x"), song("a"), song("z"), song("y")];
	queue.value = songs;
	currentIndex.value = 3;
	remember(songs[1]);

	expect(unwind(songs)).toBe(1);
	// Once the trail is spent there is nothing behind the playhead.
	expect(unwind(songs)).toBeNull();
});

test("unwind drops tracks the queue no longer holds", () => {
	const songs = [song("a"), song("b")];
	remember(song("gone"));
	remember(song("a"));

	expect(unwind(songs)).toBe(0);
	expect(unwind(songs)).toBeNull();
});

test("startTrail keeps the trail across another track of the same list", () => {
	const songs = [song("a"), song("b"), song("c")];
	queue.value = songs;
	remember(songs[0]);

	startTrail([songs[2], songs[0], songs[1]]);
	expect(unwind(songs)).toBe(0);
});

test("startTrail drops the trail when another list takes over", () => {
	queue.value = [song("a"), song("b")];
	remember(song("a"));

	startTrail([song("c")]);
	expect(unwind([song("c")])).toBeNull();
});

test("cycleRepeat walks off, all, one and back", () => {
	setRepeat("off");
	cycleRepeat();
	expect(repeat.value).toBe("all");
	cycleRepeat();
	expect(repeat.value).toBe("one");
	cycleRepeat();
	expect(repeat.value).toBe("off");
});

test("clampIndex keeps the position inside the queue", () => {
	expect(clampIndex(9, 3)).toBe(2);
	expect(clampIndex(1, 3)).toBe(1);
	// No queue, or nothing loaded, means no current track at all.
	expect(clampIndex(4, 0)).toBe(-1);
	expect(clampIndex(-1, 3)).toBe(-1);
});

test("replaceQueue pulls an out-of-range position back in", () => {
	queue.value = [song("a"), song("b"), song("c"), song("d")];
	currentIndex.value = 3;

	// A room snapshot can carry a shorter queue than the local one.
	replaceQueue([song("a")]);

	expect(currentIndex.value).toBe(0);
	expect(queue.value).toHaveLength(1);
});

test("replaceQueue keeps the position when the new queue is long enough", () => {
	queue.value = [song("a")];
	currentIndex.value = 0;

	replaceQueue([song("a"), song("b"), song("c")]);

	expect(currentIndex.value).toBe(0);
});

test("replaceQueue follows the playing track when it shifts", () => {
	queue.value = [song("a"), song("b"), song("c")];
	currentIndex.value = 2;

	replaceQueue([song("x"), song("a"), song("b"), song("c")], "c");

	expect(currentIndex.value).toBe(3);
	expect(queue.value[currentIndex.value].id).toBe("c");
});

test("skip forward availability follows the position in the queue", () => {
	setRepeat("off");
	queue.value = [song("a"), song("b"), song("c")];
	currentIndex.value = 0;
	expect(canNext.value).toBe(true);

	currentIndex.value = 2;
	expect(canNext.value).toBe(false);

	currentIndex.value = -1;
	expect(canNext.value).toBe(false);
});

test("repeat keeps the skip button enabled at the queue boundary", () => {
	queue.value = [song("a"), song("b"), song("c")];
	setRepeat("all");

	currentIndex.value = 2;
	expect(canNext.value).toBe(true);

	currentIndex.value = 0;
	expect(canNext.value).toBe(true);
});

test("skip forward stays off when there is nothing to play", () => {
	queue.value = [];
	setRepeat("all");
	currentIndex.value = -1;
	expect(canNext.value).toBe(false);
});
