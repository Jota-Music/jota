import { GetArtist, GetArtistDiscography } from "@bindings/app";
import type { ArtistDiscography, ArtistInfo } from "@/lib/music/model";

export async function getArtist(uri: string): Promise<ArtistInfo> {
	try {
		return (await GetArtist(uri)) as unknown as ArtistInfo;
	} catch (error) {
		console.error(error);
		throw error;
	}
}

export async function getArtistDiscography(
	uri: string,
): Promise<ArtistDiscography> {
	try {
		return (await GetArtistDiscography(uri)) as unknown as ArtistDiscography;
	} catch (error) {
		console.error(error);
		throw error;
	}
}
