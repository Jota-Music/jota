import { get } from "@/lib/shared/api";

export type SearchResult = {
	uri: string;
	name: string;
	type: string;
	coverUrl?: string;
	artists?: string[];
	ownerName?: string;
	trackCount?: number;
};

export async function searchSpotify(
	type: "user" | "track" | "album" | "playlist" | "artist",
	query: string,
): Promise<SearchResult[]> {
	try {
		const results = await get<SearchResult[]>(`/music/search/${type}/${encodeURIComponent(query)}`);
		return results;
	} catch (error) {
		console.error(error);
		throw error;
	}
}