import { getAlbumTracks } from "@/lib/music/app/get-album";
import { getFullPlaylist } from "@/lib/music/app/get-playlist";
import { playList } from "@/lib/music/views/stores/player";

export async function playPlaylist(id: string): Promise<void> {
	const playlist = await getFullPlaylist(id);
	const songs = playlist.songs.filter((song) => !song.broken);
	if (songs.length > 0) playList(songs, id);
}

export async function playAlbum(id: string): Promise<void> {
	const songs = (await getAlbumTracks(id)).filter((song) => !song.broken);
	if (songs.length > 0) playList(songs, id);
}
