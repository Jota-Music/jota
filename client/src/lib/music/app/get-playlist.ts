import type { Playlist } from "@/lib/music/model";
import { GetFullPlaylist, GetPlaylist } from "@/wailsjs/go/app/App";

export type { Playlist };

export default async function getPlaylist(
	id: string,
	page: number = 1,
	size: number = 15,
): Promise<Playlist> {
	try {
		return (await GetPlaylist(id, page, size)) as unknown as Playlist;
	} catch (error) {
		console.error(error);
		throw error;
	}
}

export async function getFullPlaylist(id: string): Promise<Playlist> {
	try {
		return (await GetFullPlaylist(id)) as unknown as Playlist;
	} catch (error) {
		console.error(error);
		throw error;
	}
}
