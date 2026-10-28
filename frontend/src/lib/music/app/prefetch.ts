import { getAlbumTracks } from "@/lib/music/app/get-album";
import { getArtist, getArtistDiscography } from "@/lib/music/app/get-artist";
import { getFullPlaylist } from "@/lib/music/app/get-playlist";
import { queryClient } from "@/lib/shared/views/ui/query-client";

type Load = { key: unknown[]; run: () => Promise<unknown> };

// A tile or a row names its route, and the route is what says which query the
// page will ask for, so pointing at it starts that request under the same key
// the page reads. The entry is still fresh for the default staleTime, which is
// what keeps the page from showing a skeleton for data already on its way.
const routes: Record<string, (id: string) => Load[]> = {
	playlist: (id) => [{ key: ["playlist", id], run: () => getFullPlaylist(id) }],
	album: (id) => [{ key: ["album-tracks", id], run: () => getAlbumTracks(id) }],
	artist: (id) => [
		{ key: ["artist", id], run: () => getArtist(id) },
		{ key: ["artist-discography", id], run: () => getArtistDiscography(id) },
	],
};

export function prefetch(path: string): void {
	const [route, id] = path.split("/").filter(Boolean);
	if (!route || !id) return;
	for (const load of routes[route]?.(id) ?? []) {
		void queryClient.prefetchQuery({ queryKey: load.key, queryFn: load.run });
	}
}
