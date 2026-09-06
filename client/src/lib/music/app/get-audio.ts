import type { Audio, Song } from "@/lib/music/model";
import { GetYouTubeAudio, SetYouTubeId } from "@/wailsjs/go/app/App";

export type { Audio };

export async function getAudio(
	song: Song,
): Promise<Audio & { youtube: string }> {
	try {
		const data = (await GetYouTubeAudio(
			song.id,
			`${song.name} ${song.artists.map((artist: { name: string }) => artist.name).join(", ")}`,
		)) as unknown as Audio;
		return {
			url: data.url,
			duration: data.duration,
			expireAt: data.expireAt,
			videoId: data.videoId,
			youtube: data.videoId ?? song.id,
		};
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
		await SetYouTubeId(songId, youtubeId);
	} catch (error) {
		console.error("Error updating YouTube ID:", error);
		throw error;
	}
}
