import { signal } from "@preact/signals";
import { getAlbumTracks } from "@/lib/music/app/get-album";
import { getFullPlaylist } from "@/lib/music/app/get-playlist";
import { playList } from "@/lib/music/views/stores/player";

export const loadingPlaylist = signal<string | null>(null);

export async function playPlaylist(id: string): Promise<void> {
	loadingPlaylist.value = id;
	try {
		const playlist = await getFullPlaylist(id);
		const songs = playlist.songs.filter((song) => !song.broken);
		if (songs.length > 0) playList(songs, id);
	} finally {
		loadingPlaylist.value = null;
	}
}

export async function playAlbum(id: string): Promise<void> {
	const songs = (await getAlbumTracks(id)).filter((song) => !song.broken);
	if (songs.length > 0) playList(songs, id);
}
