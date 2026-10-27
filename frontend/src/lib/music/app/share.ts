import { playlistSource } from "@/lib/music/app/playlist-source";
import type { Song } from "@/lib/music/model";

export function spotifyLink(song: Song): string | null {
	if (playlistSource(song.id) !== "spotify") return null;
	return song.share?.url || null;
}

export function youtubeLink(song: Song): string | null {
	return song.youtubeId
		? `https://www.youtube.com/watch?v=${song.youtubeId}`
		: null;
}

export function canShare(song: Song): boolean {
	return playlistSource(song.id) !== "local";
}

// Route ids are bare for Spotify entities, `youtube:`-prefixed for YouTube
// ones, and `local:` or `...:collection` for things that exist only in the
// store, so they get no link.
const SPOTIFY_ID = /^[0-9a-zA-Z]{21,22}$/;

export function pageLink(
	page: "playlist" | "album" | "artist",
	id: string,
): string | null {
	if (page === "playlist" && id.startsWith("youtube:")) {
		return `https://www.youtube.com/playlist?list=${id.slice("youtube:".length)}`;
	}
	const spotifyId = id.startsWith("spotify:")
		? id.slice("spotify:".length)
		: id;
	if (!SPOTIFY_ID.test(spotifyId)) return null;
	return `https://open.spotify.com/${page}/${spotifyId}`;
}
