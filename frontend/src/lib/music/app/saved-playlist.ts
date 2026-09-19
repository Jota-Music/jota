import { GetSavedPlaylists, RemoveSavedPlaylist } from "@bindings/app";
import type { PlaylistSummary } from "@/lib/music/model";

export async function getSavedPlaylists(): Promise<PlaylistSummary[]> {
	return (await GetSavedPlaylists()) ?? [];
}

export async function removeSavedPlaylist(id: string): Promise<void> {
	await RemoveSavedPlaylist(id);
}
