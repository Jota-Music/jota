import { GetFullPlaylist } from "@bindings/app";
import type { Playlist } from "@/lib/music/model";

export async function getFullPlaylist(id: string): Promise<Playlist> {
	try {
		return (await GetFullPlaylist(id)) as unknown as Playlist;
	} catch (error) {
		console.error(error);
		throw error;
	}
}
