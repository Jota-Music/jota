import type { Song } from "@/lib/music/model";
import { get } from "@/lib/shared/api";

export async function getSong(id: string): Promise<Song> {
	try {
		const response = await get<Song>(`/music/song/${id}`);
		return response;
	} catch (error) {
		console.error(error);
		throw error;
	}
}
