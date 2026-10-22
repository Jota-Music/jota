import { expect, test } from "bun:test";
import { canShare, spotifyLink, youtubeLink } from "@/lib/music/app/share";
import type { Song } from "@/lib/music/model";

function track(id: string, extra: Partial<Song> = {}): Song {
	return {
		id,
		url: "",
		name: "Rambo",
		duration: 182,
		share: { id, url: `https://open.spotify.com/track/${id}` },
		album: { id: "", title: "", url: "", covers: [] },
		artists: [],
		...extra,
	};
}

test("links a Spotify track to its share url", () => {
	expect(spotifyLink(track("4cOdK2wGLETKBW3PvgPWqT"))).toBe(
		"https://open.spotify.com/track/4cOdK2wGLETKBW3PvgPWqT",
	);
});

// A YouTube or local song has no Spotify page to point at, even if a share
// payload came along with it.
test("drops the Spotify link for non-Spotify sources", () => {
	expect(spotifyLink(track("youtube:dQw4w9WgXcQ"))).toBeNull();
	expect(spotifyLink(track("local:shelf"))).toBeNull();
});

test("builds a YouTube link from the linked video", () => {
	expect(youtubeLink(track("local:shelf", { youtubeId: "dQw4w9WgXcQ" }))).toBe(
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	);
	expect(youtubeLink(track("local:shelf"))).toBeNull();
});

test("local songs cannot be shared", () => {
	expect(canShare(track("local:shelf"))).toBe(false);
	expect(canShare(track("4cOdK2wGLETKBW3PvgPWqT"))).toBe(true);
	expect(canShare(track("youtube:PL1234567890abcdef"))).toBe(true);
});
