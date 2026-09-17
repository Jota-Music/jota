import { beforeEach, expect, test } from "bun:test";
import type { QueueSong, Song } from "@/lib/music/model";
import {
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

function song(id: string, youtubeId?: string, queued = false): QueueSong {
	return { id, name: id, ...base, youtubeId, queued };
}

beforeEach(() => {
	queue.value = [];
});

test("setSongYoutubeId fills the resolved id", () => {
	queue.value = [song("a")];
	setSongYoutubeId("a", "vid");
	expect(queue.value[0].youtubeId).toBe("vid");
});

test("setSongYoutubeId keeps the queued marker", () => {
	queue.value = [song("a", undefined, true)];
	setSongYoutubeId("a", "vid");
	expect(queue.value[0]).toMatchObject({ youtubeId: "vid", queued: true });
});

test("setSongYoutubeId ignores unknown songs and no-op updates", () => {
	queue.value = [song("a", "vid")];
	setSongYoutubeId("missing", "x");
	setSongYoutubeId("a", "vid");
	expect(queue.value).toHaveLength(1);
	expect(queue.value[0].youtubeId).toBe("vid");
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
