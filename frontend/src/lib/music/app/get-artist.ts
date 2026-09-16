import { GetArtist, GetArtistDiscography } from "@bindings/app";
import type { ArtistDiscography, ArtistInfo } from "@/lib/music/model";

export async function getArtist(uri: string): Promise<ArtistInfo> {
	return (await GetArtist(uri)) as unknown as ArtistInfo;
}

export async function getArtistDiscography(
	uri: string,
): Promise<ArtistDiscography> {
	return (await GetArtistDiscography(uri)) as unknown as ArtistDiscography;
}
