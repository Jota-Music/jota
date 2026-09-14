import {
	AddYouTubePlaylist,
	GetYouTubePlaylists,
	RemoveYouTubePlaylist,
} from "@bindings/app";
import type { PlaylistSummary } from "@/lib/music/model";

export async function getYouTubePlaylists(): Promise<PlaylistSummary[]> {
	const playlists = (await GetYouTubePlaylists()) as unknown as
		| PlaylistSummary[]
		| null;
	return playlists ?? [];
}

export async function addYouTubePlaylist(id: string): Promise<PlaylistSummary> {
	return (await AddYouTubePlaylist(id)) as unknown as PlaylistSummary;
}

export async function removeYouTubePlaylist(id: string): Promise<void> {
	await RemoveYouTubePlaylist(id);
}
