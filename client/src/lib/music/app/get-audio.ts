import type { Audio, Song } from "@/lib/music/model";
import { get, post } from "@/lib/shared/api";

export async function getAudio(song: Song): Promise<Audio> {
	try {
		const response = await get<Audio>(`/youtube/audio/${song.id}`, {
			query: {
				search: `${song.name} ${song.artists.map((artist) => artist.name).join(", ")}`,
			},
		});
		return response;
	} catch (error) {
		console.error(error);
		throw error;
	}
}

export async function updateYoutubeId(
	songId: string,
	youtubeId: string,
): Promise<void> {
	try {
		await post(`/music/link-youtube`, { id: songId, youtubeId });
	} catch (error) {
		console.error("Error updating YouTube ID:", error);
		throw error;
	}
}
