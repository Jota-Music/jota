import { GetAlbumTracks } from "@bindings/app";
import type { Song } from "@/lib/music/model";

export async function getAlbumTracks(uri: string): Promise<Song[]> {
	return (await GetAlbumTracks(uri)) as unknown as Song[];
}
