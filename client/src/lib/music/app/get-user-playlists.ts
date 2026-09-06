import type { PlaylistSummary } from "@/lib/music/model";
import { GetUserPlaylists } from "@/wailsjs/go/app/App";

export type { PlaylistSummary };

export default async function getUserPlaylists(
	user: string,
): Promise<PlaylistSummary[]> {
	try {
		return (await GetUserPlaylists(user)) as unknown as PlaylistSummary[];
	} catch (error) {
		console.error(error);
		throw error;
	}
}
