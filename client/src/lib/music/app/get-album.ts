import type { Song } from "@/lib/music/model";
import { get } from "@/lib/shared/api";

export async function getAlbumTracks(uri: string): Promise<Song[]> {
	try {
		const response = await get<Song[]>(`/music/album/${encodeURIComponent(uri)}`);
		return response;
	} catch (error) {
		console.error(error);
		throw error;
	}
}