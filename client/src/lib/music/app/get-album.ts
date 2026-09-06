import type { Song } from "@/lib/music/model";
import { GetAlbumTracks } from "@/wailsjs/go/app/App";

export async function getAlbumTracks(uri: string): Promise<Song[]> {
	try {
		return (await GetAlbumTracks(uri)) as unknown as Song[];
	} catch (error) {
		console.error(error);
		throw error;
	}
}
