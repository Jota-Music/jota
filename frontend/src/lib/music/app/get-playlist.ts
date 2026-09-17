import { GetFullPlaylist, RevalidateFullPlaylist } from "@bindings/app";
import type { Playlist } from "@/lib/music/model";

export async function getFullPlaylist(id: string): Promise<Playlist> {
	const data = await GetFullPlaylist(id);
	return { ...data, songs: data.songs ?? [] };
}

export async function revalidateFullPlaylist(id: string): Promise<void> {
	await RevalidateFullPlaylist(id);
}
