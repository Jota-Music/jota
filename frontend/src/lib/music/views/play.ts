import { signal } from "@preact/signals";
import type { QueryClient } from "@tanstack/preact-query";
import { getAlbumTracks } from "@/lib/music/app/get-album";
import { getFullPlaylist } from "@/lib/music/app/get-playlist";
import { playList } from "@/lib/music/views/stores/player";

export const loadingPlaylist = signal<string | null>(null);

// fetchQuery goes through the cache the page already filled, so pressing play on
// an open playlist does not pull the whole thing over the bridge again. It still
// refetches when the entry was invalidated (a song was just removed) or aged out,
// so this never plays a list the UI has moved on from.
export async function playPlaylist(qc: QueryClient, id: string): Promise<void> {
	loadingPlaylist.value = id;
	try {
		const playlist = await qc.fetchQuery({
			queryKey: ["playlist", id],
			queryFn: () => getFullPlaylist(id),
		});
		const songs = playlist.songs.filter((song) => !song.broken);
		if (songs.length > 0) playList(songs, id);
	} finally {
		loadingPlaylist.value = null;
	}
}

export async function playAlbum(qc: QueryClient, id: string): Promise<void> {
	const songs = (
		await qc.fetchQuery({
			queryKey: ["album-tracks", id],
			queryFn: () => getAlbumTracks(id),
		})
	).filter((song) => !song.broken);
	if (songs.length > 0) playList(songs, id);
}
