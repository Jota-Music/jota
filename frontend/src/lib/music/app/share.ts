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
