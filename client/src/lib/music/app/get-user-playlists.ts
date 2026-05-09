import { get } from "@/lib/shared/api";

export type PlaylistSummary =
  | {
      id: string;
      name: string;
      mosaic: string;
    }
  | {
      id: string;
      name: string;
      cover: string;
    };

export default async function getUserPlaylists(
  user: string,
): Promise<PlaylistSummary[]> {
  try {
    const playlists = await get<PlaylistSummary[]>(`/music/playlists/${user}`);

    return playlists;
  } catch (error) {
    console.error(error);
    throw error;
  }
}
