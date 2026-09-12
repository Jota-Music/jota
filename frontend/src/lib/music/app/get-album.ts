import { GetAlbumTracks } from "@bindings/app";
import type { Song } from "@/lib/music/model";

export async function getAlbumTracks(uri: string): Promise<Song[]> {
	try {
		return (await GetAlbumTracks(uri)) as unknown as Song[];
	} catch (error) {
		console.error(error);
		throw error;
	}
}
