import { GetYouTubeChannelInfo, GetYouTubeChannelVideos } from "@bindings/app";
import type { ChannelInfo, Song } from "@/lib/music/model";

export async function getChannelVideos(id: string): Promise<Song[]> {
	return (await GetYouTubeChannelVideos(id)) ?? [];
}

export async function getChannelInfo(id: string): Promise<ChannelInfo> {
	return GetYouTubeChannelInfo(id);
}
