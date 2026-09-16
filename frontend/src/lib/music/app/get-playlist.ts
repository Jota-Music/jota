import { GetFullPlaylist } from "@bindings/app";
import type { Playlist } from "@/lib/music/model";

export async function getFullPlaylist(id: string): Promise<Playlist> {
	return (await GetFullPlaylist(id)) as unknown as Playlist;
}
