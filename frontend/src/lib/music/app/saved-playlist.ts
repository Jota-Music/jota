import {
	AddSavedPlaylist,
	GetSavedPlaylists,
	RemoveSavedPlaylist,
} from "@bindings/app";
import type { PlaylistSummary } from "@/lib/music/model";

export async function getSavedPlaylists(): Promise<PlaylistSummary[]> {
	return (await GetSavedPlaylists()) ?? [];
}

export async function addSavedPlaylist(id: string): Promise<void> {
	await AddSavedPlaylist(id);
}

export async function removeSavedPlaylist(id: string): Promise<void> {
	await RemoveSavedPlaylist(id);
}
