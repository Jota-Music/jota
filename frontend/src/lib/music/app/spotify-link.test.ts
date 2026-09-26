import { expect, test } from "bun:test";
import { parseSpotifyLink, spotifyRef } from "@/lib/music/app/spotify-link";

const ARTIST = { type: "artist", id: "4Z8W4fKeB5YxbusRsdQVPb" } as const;
const TRACK = { type: "track", id: "4cOdK2wGLETKBW3PvgPWqT" } as const;

test("parses artist links, URIs and bare ids", () => {
	expect(parseSpotifyLink(`spotify:artist:${ARTIST.id}`)).toEqual(ARTIST);
	expect(
		parseSpotifyLink(`https://open.spotify.com/artist/${ARTIST.id}`),
	).toEqual(ARTIST);
	expect(
		parseSpotifyLink(`https://open.spotify.com/intl-es/artist/${ARTIST.id}`),
	).toEqual(ARTIST);
	expect(parseSpotifyLink(ARTIST.id)).toBeNull();
	expect(parseSpotifyLink(`spotify:track:${TRACK.id}`)).toEqual(TRACK);
});

// A bare name is not a reference. Treating it as one sent the search page to a
// detail page that could never resolve it.
test("rejects names that are not base62 ids", () => {
	expect(parseSpotifyLink("Radiohead")).toBeNull();
	expect(parseSpotifyLink("spotify:artist:Radiohead")).toBeNull();
	expect(
		parseSpotifyLink("https://open.spotify.com/artist/Radiohead"),
	).toBeNull();
	expect(parseSpotifyLink("spotify:artist:")).toBeNull();
	expect(parseSpotifyLink("spotify:nonsense:aaaaaaaaaaaaaaaaaaaaa")).toBeNull();
	expect(parseSpotifyLink("")).toBeNull();
});

// Usernames are textual, unlike every other entity id.
test("accepts textual usernames", () => {
	expect(parseSpotifyLink("spotify:user:jota")).toEqual({
		type: "user",
		id: "jota",
	});
	expect(parseSpotifyLink("https://open.spotify.com/user/jota")).toEqual({
		type: "user",
		id: "jota",
	});
});

// The search page receives a bare id: the header already resolved the pasted
// link and left the type in the route.
test("resolves a bare id against the search type", () => {
	const id = "5wC2X3QsfCc8M3KVv0wkDa";
	expect(spotifyRef("playlist", id)).toEqual({ type: "playlist", id });
	expect(spotifyRef("album", ` ${id} `)).toEqual({ type: "album", id });
	expect(spotifyRef("playlist", "Radiohead")).toBeNull();
	expect(spotifyRef("nonsense", id)).toBeNull();
	expect(
		spotifyRef("playlist", `https://open.spotify.com/artist/${ARTIST.id}`),
	).toEqual(ARTIST);
});
