import { Search, SearchYouTube, SearchYouTubePlaylists } from "@bindings/app";
import type { PlaylistSummary, SearchResult, Song } from "@/lib/music/model";

export type { SearchResult } from "@/lib/music/model";

export async function searchSpotify(
	type_: "user" | "track" | "album" | "playlist" | "artist",
	query: string,
): Promise<SearchResult[]> {
	return (await Search(query, type_)) ?? [];
}

interface YouTubeVideo {
	id: string;
	title: string;
}

export async function searchYouTube(query: string): Promise<YouTubeVideo[]> {
	const results = (await SearchYouTube(query)) ?? [];
	return results.map((v) => ({ id: v.id, title: v.title }));
}

export async function searchYouTubePlaylists(
	query: string,
): Promise<PlaylistSummary[]> {
	return (await SearchYouTubePlaylists(query)) ?? [];
}

export function youtubeVideoToSong(video: YouTubeVideo): Song {
	const id = `youtube:${video.id}`;
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
		artists: [{ name: "YouTube" }],
		youtubeId: video.id,
	};
}
