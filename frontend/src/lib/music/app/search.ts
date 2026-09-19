import { Search, SearchYouTube, SearchYouTubePlaylists } from "@bindings/app";
import type { Video } from "@models/services/youtube/models";
import type { PlaylistSummary, SearchResult, Song } from "@/lib/music/model";

export type { SearchResult } from "@/lib/music/model";

export async function searchSpotify(
	type_: "user" | "track" | "album" | "playlist" | "artist",
	query: string,
): Promise<SearchResult[]> {
	return (await Search(query, type_)) ?? [];
}

export async function searchYouTube(query: string): Promise<Video[]> {
	return (await SearchYouTube(query)) ?? [];
}

export async function searchYouTubePlaylists(
	query: string,
): Promise<PlaylistSummary[]> {
	return (await SearchYouTubePlaylists(query)) ?? [];
}

export function youtubeVideoToSong(video: Video): Song {
	const id = `youtube:${video.id}`;
	const author = video.author || "YouTube";
	const thumbnailUrl = `https://img.youtube.com/vi/${video.id}/default.jpg`;
	return {
		id,
		name: video.title,
		duration: 0,
		url: "",
		share: {
			id,
			url: `https://www.youtube.com/watch?v=${video.id}`,
		},
		album: {
			id: "",
			title: "YouTube",
			url: "",
			covers: [thumbnailUrl],
		},
		artists: [{ id: video.channelId, name: author, source: "youtube" }],
		youtubeId: video.id,
	};
}
