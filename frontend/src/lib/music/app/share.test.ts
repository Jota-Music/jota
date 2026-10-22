import { expect, test } from "bun:test";
import {
	canShare,
	pageLink,
	spotifyLink,
	youtubeLink,
} from "@/lib/music/app/share";
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

test("links a page to its source page", () => {
	expect(pageLink("playlist", "37i9dQZF1DXcBWIGoYBM5M")).toBe(
		"https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M",
	);
	expect(pageLink("album", "4aawyAB9vmqN3uQ7FjRGTy")).toBe(
		"https://open.spotify.com/album/4aawyAB9vmqN3uQ7FjRGTy",
	);
	expect(pageLink("artist", "0OdUWJ0sBjDrqHygGUXeCF")).toBe(
		"https://open.spotify.com/artist/0OdUWJ0sBjDrqHygGUXeCF",
	);
	expect(pageLink("playlist", "youtube:PL1234567890abcdef")).toBe(
		"https://www.youtube.com/playlist?list=PL1234567890abcdef",
	);
});

// A local playlist lives in the store and Liked Songs is a context, so neither
// has a page to hand out.
test("local and likedsongs pages have no link", () => {
	expect(pageLink("playlist", "local:7f3a")).toBeNull();
	expect(pageLink("playlist", "spotify:user:me:collection")).toBeNull();
});
