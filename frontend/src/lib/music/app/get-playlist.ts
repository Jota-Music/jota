import { GetFullPlaylist, RevalidateFullPlaylist } from "@bindings/app";
import type { Playlist } from "@/lib/music/model";

export async function getFullPlaylist(id: string): Promise<Playlist> {
	return (await GetFullPlaylist(id)) as unknown as Playlist;
}

export async function revalidateFullPlaylist(id: string): Promise<void> {
	await RevalidateFullPlaylist(id);
}
