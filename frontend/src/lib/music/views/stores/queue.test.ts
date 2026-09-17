import { beforeEach, expect, test } from "bun:test";
import { permute, random } from "@/lib/music/app/shuffle";
import type { Song } from "@/lib/music/model";
import {
	applyShuffle,
	currentIndex,
	cycleRepeat,
	queue,
	repeat,
	setRepeat,
	setSongYoutubeId,
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

test("cycleRepeat walks off, all, one and back", () => {
	setRepeat("off");
	cycleRepeat();
	expect(repeat.value).toBe("all");
	cycleRepeat();
	expect(repeat.value).toBe("one");
	cycleRepeat();
	expect(repeat.value).toBe("off");
});
