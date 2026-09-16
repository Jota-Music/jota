import { GetUserPlaylists, RevalidateUserPlaylists } from "@bindings/app";
import type { PlaylistSummary } from "@/lib/music/model";

export default async function getUserPlaylists(
	user: string,
): Promise<PlaylistSummary[]> {
	return (await GetUserPlaylists(user)) as unknown as PlaylistSummary[];
}

export async function revalidateUserPlaylists(user: string): Promise<void> {
	await RevalidateUserPlaylists(user);
}
