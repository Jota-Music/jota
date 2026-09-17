import { ResolveAudio, SetYouTubeId } from "@bindings/app";
import type { Audio, Song } from "@/lib/music/model";

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
	const data = await ResolveAudio(song);
	return toAudio(data, song.youtubeId ?? song.id);
}

export async function updateYoutubeId(
	songId: string,
	youtubeId: string,
): Promise<void> {
	await SetYouTubeId(songId, youtubeId);
}
