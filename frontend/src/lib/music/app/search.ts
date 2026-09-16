import { Search, SearchYouTube, SearchYouTubePlaylists } from "@bindings/app";
import type { PlaylistSummary, Song } from "@/lib/music/model";

export type SearchResult = {
	uri: string;
	name: string;
	type: string;
	coverUrl?: string;
	artists?: string[];
	ownerName?: string;
	trackCount?: number;
};

export async function searchSpotify(
	type_: "user" | "track" | "album" | "playlist" | "artist",
	query: string,
): Promise<SearchResult[]> {
	return (await Search(query, type_)) as unknown as SearchResult[];
}

export interface YouTubeVideo {
	id: string;
	title: string;
}

export async function searchYouTube(query: string): Promise<YouTubeVideo[]> {
	const results = (await SearchYouTube(query)) as unknown as Array<{
		ID: string;
		Title: string;
	}>;
	return results.map((r) => ({ id: r.ID, title: r.Title }));
}

export async function searchYouTubePlaylists(
	query: string,
): Promise<PlaylistSummary[]> {
	const results = (await SearchYouTubePlaylists(query)) as unknown as
		| PlaylistSummary[]
		| null;
	return results ?? [];
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
			title: "YouTube",
			url: "",
			covers: [thumbnailUrl],
		},
		artists: [{ name: "YouTube" }],
		youtubeId: video.id,
	};
}
