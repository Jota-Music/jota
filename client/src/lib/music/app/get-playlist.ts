import type { Playlist } from "@/lib/music/model";
import { get } from "@/lib/shared/api";

export default async function getPlaylist(
    id: string,
    page: number = 1,
    size: number = 15
): Promise<Playlist> {
    try {
        const response = await get<Playlist>(`/music/playlist/${id}`, {
            query: {
                page: page.toString(),
                size: size.toString(),
            },
        });
        return response;
    } catch (error) {
        console.error(error);
        throw error;
    }
}

export async function getFullPlaylist(id: string): Promise<Playlist> {
    try {
        const response = await get<Playlist>(`/music/playlist/full/${id}`);
        return response;
    } catch (error) {
        console.error(error);
        throw error;
    }
}