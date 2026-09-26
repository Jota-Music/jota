import {
	Search,
	SearchYouTube,
	SearchYouTubeChannels,
	SearchYouTubePlaylists,
} from "@bindings/app";
import type { Video } from "@models/services/youtube/models";
import type {
	ChannelInfo,
	PlaylistSummary,
	SearchResult,
	Song,
} from "@/lib/music/model";

export type { SearchResult } from "@/lib/music/model";

export function stripUriPrefix(query: string, type: string): string {
	const prefix = `spotify:${type}:`;
	return query.startsWith(prefix) ? query.slice(prefix.length) : query;
}

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

export async function searchYouTubeChannels(
	query: string,
): Promise<ChannelInfo[]> {
	return (await SearchYouTubeChannels(query)) ?? [];
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

export function searchResultToSong(item: SearchResult): Song {
	const id = stripUriPrefix(item.uri, "track");

	return {
		id,
		name: item.name,
		duration: item.duration ?? 0,
		url: "",
		share: {
			id,
			url: `https://open.spotify.com/track/${id}`,
		},
		album: {
			id: "",
			title: "",
			url: "",
			covers: item.coverUrl ? [item.coverUrl] : [],
		},
		artists: item.artists ?? [],
	};
}
