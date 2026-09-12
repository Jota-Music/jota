import { GetUserPlaylists } from "@bindings/app";
import type { PlaylistSummary } from "@/lib/music/model";

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
