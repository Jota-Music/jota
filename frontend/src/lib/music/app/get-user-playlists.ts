import {
	GetUserPlaylists,
	GetYouTubeChannelPlaylists,
	RevalidateUserPlaylists,
	RevalidateYouTubeChannel,
} from "@bindings/app";
import type { PlaylistSummary } from "@/lib/music/model";

export type UserSource = "spotify" | "youtube";

export async function getUserPlaylists(
	source: UserSource,
	id: string,
): Promise<PlaylistSummary[]> {
	const playlists =
		source === "youtube"
			? await GetYouTubeChannelPlaylists(id)
			: await GetUserPlaylists(id);
	return playlists ?? [];
}

export async function revalidateUserPlaylists(
	source: UserSource,
	id: string,
): Promise<void> {
	if (source === "youtube") {
		await RevalidateYouTubeChannel(id);
		return;
	}
	await RevalidateUserPlaylists(id);
}
