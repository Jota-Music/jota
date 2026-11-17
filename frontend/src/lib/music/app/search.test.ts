import { expect, test } from "bun:test";
import {
	type SearchResult,
	searchResultToSong,
	youtubeVideoToSong,
} from "@/lib/music/app/search";

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

// The channel id is the same in every client, and the app's own form carries
// the prefix; a result that only names the channel still becomes that id.
test("names the channel the way the app stores it", () => {
	const song = youtubeVideoToSong({
		id: "dQw4w9WgXcQ",
		title: "Never Gonna Give You Up",
		author: "Rick Astley",
		channelId: "UCuAXFkgsw1L7xaCfnd5JJOw",
	});

	expect(song.artists).toEqual([
		{
			id: "youtube:channel:UCuAXFkgsw1L7xaCfnd5JJOw",
			name: "Rick Astley",
			source: "youtube",
		},
	]);
});

test("leaves the channel empty when the result names none", () => {
	const song = youtubeVideoToSong({ id: "dQw4w9WgXcQ", title: "Video" });

	expect(song.artists).toEqual([
		{ id: "", name: "YouTube", source: "youtube" },
	]);
});
