import {
	AddSongsToPlaylist,
	CreatePlaylist,
	DeletePlaylist,
	GetPlaylists,
	RemoveSongFromPlaylist,
	ReorderPlaylist,
} from "@bindings/app";
import type { PlaylistSummary } from "@/lib/music/model";

const LOCAL_PREFIX = "local:";

export function isCustom(id: string): boolean {
	return id.startsWith(LOCAL_PREFIX);
}

export async function getPlaylists(): Promise<PlaylistSummary[]> {
	return (await GetPlaylists()) ?? [];
}

export async function createPlaylist(name: string): Promise<PlaylistSummary> {
	return CreatePlaylist(name);
}

export async function deletePlaylist(id: string): Promise<void> {
	await DeletePlaylist(id);
}

export async function addSongsToPlaylist(
	id: string,
	refs: string[],
): Promise<void> {
	await AddSongsToPlaylist(id, refs);
}

export async function removeSongFromPlaylist(
	id: string,
	ref: string,
): Promise<void> {
	await RemoveSongFromPlaylist(id, ref);
}

export async function reorderPlaylist(
	id: string,
	refs: string[],
): Promise<void> {
	await ReorderPlaylist(id, refs);
}
