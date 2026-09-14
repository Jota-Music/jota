import { ResolveAudio, SetYouTubeId } from "@bindings/app";
import type { Audio, Song } from "@/lib/music/model";

export type { Audio };

function toAudio(data: Audio, fallback: string): Audio & { youtube: string } {
	return {
		url: data.url,
		duration: data.duration,
		expireAt: data.expireAt,
		videoId: data.videoId,
		youtube: data.videoId ?? fallback,
	};
}

export async function getAudio(
	song: Song,
): Promise<Audio & { youtube: string }> {
	try {
		const data = (await ResolveAudio(
			song as unknown as Parameters<typeof ResolveAudio>[0],
		)) as unknown as Audio;
		return toAudio(data, song.youtubeId ?? song.id);
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
