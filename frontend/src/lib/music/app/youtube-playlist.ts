import {
	AddYouTubePlaylist,
	GetYouTubePlaylists,
	RemoveYouTubePlaylist,
} from "@bindings/app";
import type { PlaylistSummary } from "@/lib/music/model";

export async function getYouTubePlaylists(): Promise<PlaylistSummary[]> {
	return (await GetYouTubePlaylists()) ?? [];
}

export async function addYouTubePlaylist(id: string): Promise<PlaylistSummary> {
	return AddYouTubePlaylist(id);
}

export async function removeYouTubePlaylist(id: string): Promise<void> {
	await RemoveYouTubePlaylist(id);
}
