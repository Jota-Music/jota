import { GetSong } from "@bindings/app";
import type { Song } from "@/lib/music/model";

export async function getSong(id: string): Promise<Song> {
	const song = await GetSong(id);
	return { ...song, artists: song.artists ?? [] };
}
