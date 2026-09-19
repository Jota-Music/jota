import { expect, test } from "bun:test";
import { parseYoutubeLink } from "@/lib/music/app/youtube-link";

const VIDEO = { type: "video", id: "dQw4w9WgXcQ" } as const;
const PLAYLIST = { type: "playlist", id: "PL1234567890abcdef" } as const;
const CHANNEL_ID = "UC-lHJZR3Gqxm24_Vd_AJ5Yw";
const CHANNEL = { type: "channel", id: CHANNEL_ID } as const;

test("parses video links and bare IDs", () => {
	expect(
		parseYoutubeLink("https://www.youtube.com/watch?v=dQw4w9WgXcQ"),
	).toEqual(VIDEO);
	expect(parseYoutubeLink("https://youtu.be/dQw4w9WgXcQ")).toEqual(VIDEO);
	expect(
		parseYoutubeLink("https://music.youtube.com/watch?v=dQw4w9WgXcQ"),
	).toEqual(VIDEO);
	expect(
		parseYoutubeLink("https://www.youtube.com/shorts/dQw4w9WgXcQ"),
	).toEqual(VIDEO);
	expect(parseYoutubeLink("https://www.youtube.com/embed/dQw4w9WgXcQ")).toEqual(
		VIDEO,
	);
	expect(parseYoutubeLink("https://www.youtube.com/live/dQw4w9WgXcQ")).toEqual(
		VIDEO,
	);
	expect(parseYoutubeLink("dQw4w9WgXcQ")).toEqual(VIDEO);
});

test("a video wins over a playlist", () => {
	expect(
		parseYoutubeLink(
			"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL1234567890abcdef",
		),
	).toEqual(VIDEO);
});

test("parses playlist links and bare IDs", () => {
	expect(
		parseYoutubeLink(
			"https://www.youtube.com/playlist?list=PL1234567890abcdef",
		),
	).toEqual(PLAYLIST);
	expect(parseYoutubeLink("PL1234567890abcdef")).toEqual(PLAYLIST);
});

test("parses channel links, handles and bare IDs", () => {
	expect(parseYoutubeLink(CHANNEL_ID)).toEqual(CHANNEL);
	expect(parseYoutubeLink("@PewDiePie")).toEqual({
		type: "channel",
		id: "@PewDiePie",
	});
	expect(parseYoutubeLink("https://www.youtube.com/@PewDiePie")).toEqual({
		type: "channel",
		id: "@PewDiePie",
	});
	expect(
		parseYoutubeLink(`https://www.youtube.com/channel/${CHANNEL_ID}`),
	).toEqual(CHANNEL);
	expect(parseYoutubeLink("https://www.youtube.com/c/PewDiePie")).toEqual({
		type: "channel",
		id: "https://www.youtube.com/c/PewDiePie",
	});
});

test("rejects non-YouTube input", () => {
	expect(parseYoutubeLink("jazz music")).toBeNull();
	expect(parseYoutubeLink("https://open.spotify.com/track/abc")).toBeNull();
	expect(parseYoutubeLink("")).toBeNull();
});
