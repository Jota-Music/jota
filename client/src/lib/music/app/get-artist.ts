import type { ArtistDiscography, ArtistInfo } from "@/lib/music/model";
import { get } from "@/lib/shared/api";

export async function getArtist(uri: string): Promise<ArtistInfo> {
	try {
		const response = await get<ArtistInfo>(
			`/music/artist/${encodeURIComponent(uri)}`,
		);
		return response;
	} catch (error) {
		console.error(error);
		throw error;
	}
}

export async function getArtistDiscography(
	uri: string,
): Promise<ArtistDiscography> {
	try {
		const response = await get<ArtistDiscography>(
			`/music/artist/${encodeURIComponent(uri)}/discography`,
		);
		return response;
	} catch (error) {
		console.error(error);
		throw error;
	}
}
