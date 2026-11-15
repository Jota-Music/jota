import { GetMyYouTubePlaylists } from "@bindings/app";
import { getAlbumTracks } from "@/lib/music/app/get-album";
import { getArtist, getArtistDiscography } from "@/lib/music/app/get-artist";
import { getFullPlaylist } from "@/lib/music/app/get-playlist";
import { getUserPlaylists } from "@/lib/music/app/get-user-playlists";
import { getPlaylists } from "@/lib/music/app/playlists";
import { getSavedPlaylists } from "@/lib/music/app/saved-playlist";
import { getOrder } from "@/lib/shared/app/order";
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

// The library the app opens on. Both reads are answered before the first
// render, so the shelf paints its whole set at once: filling in as each source
// lands read as the library growing a tile per frame.
export function warm(): Promise<void> {
	return Promise.all([
		queryClient.prefetchQuery({
			queryKey: ["playlists"],
			queryFn: getPlaylists,
		}),
		queryClient.prefetchQuery({
			queryKey: ["saved-playlists"],
			queryFn: getSavedPlaylists,
		}),
	]).then(() => undefined);
}

// The order the shelf paints in, which every install keeps. It is read before
// the first render because a shelf that reorders once the order lands is a shelf
// that jumps: the read is the store in memory, so it costs a bridge call.
export function warmOrder(account: string | null): Promise<void> {
	return queryClient
		.prefetchQuery({
			queryKey: ["shelf-order", "library", account ?? "default"],
			queryFn: () => getOrder("library", account ?? "default"),
		})
		.then(() => undefined);
}

// The lists a source owns are the one part of the shelf that can need the
// network, on a cache the source has to fill. They are asked together and given
// one short wait, not the launch: a cache that answers paints the shelf whole,
// and one the source has to fetch lands on the library this device holds.
export function warmSources(
	account: string | null,
	youtube: boolean,
): Promise<void> {
	const asks: Promise<unknown>[] = [];
	if (account) {
		asks.push(
			queryClient.prefetchQuery({
				queryKey: ["user-playlists", "spotify", account],
				queryFn: () => getUserPlaylists("spotify", account),
			}),
		);
	}
	// The YouTube account's list sits in the same shelf, under a key that names
	// no account because one install holds one session.
	if (youtube) {
		asks.push(
			queryClient.prefetchQuery({
				queryKey: ["user-playlists", "youtube"],
				queryFn: () => GetMyYouTubePlaylists(),
			}),
		);
	}
	return Promise.all(asks).then(() => undefined);
}
