import { expect, test } from "bun:test";
import { type SearchResult, searchResultToSong } from "@/lib/music/app/search";

// The search payload used to arrive without durations or artist ids, so the
// mapper silently produced 00:00 rows and unclickable artist names.
test("passes duration and artist ids through to the song", () => {
	const item: SearchResult = {
		uri: "spotify:track:4cOdK2wGLETKBW3PvgPWqT",
		name: "Creep",
		type: "track",
		coverUrl: "https://cover",
		artists: [{ id: "4Z8W4fKeB5YxbusRsdQVPb", name: "Radiohead" }],
		duration: 238,
	};

	const song = searchResultToSong(item);

	expect(song.id).toBe("4cOdK2wGLETKBW3PvgPWqT");
	expect(song.duration).toBe(238);
	expect(song.artists).toEqual([
		{ id: "4Z8W4fKeB5YxbusRsdQVPb", name: "Radiohead" },
	]);
});

test("defaults duration, artists and covers when absent", () => {
	const song = searchResultToSong({
		uri: "spotify:track:x",
		name: "n",
		type: "track",
	});

	expect(song.duration).toBe(0);
	expect(song.artists).toEqual([]);
	expect(song.album.covers).toEqual([]);
});
