import { GetArtist, GetArtistDiscography } from "@bindings/app";
import type { ArtistDiscography, ArtistInfo } from "@/lib/music/model";

export async function getArtist(uri: string): Promise<ArtistInfo> {
	const data = await GetArtist(uri);
	return { ...data, tracks: data.tracks ?? [] };
}

export async function getArtistDiscography(
	uri: string,
): Promise<ArtistDiscography> {
	const data = await GetArtistDiscography(uri);
	return { ...data, albums: data.albums ?? [] };
}
