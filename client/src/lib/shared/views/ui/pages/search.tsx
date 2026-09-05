import { useQuery } from "@tanstack/preact-query";
import { Loader } from "lucide-preact";
import { useLayoutEffect } from "preact/hooks";
import { Link, useLocation, useRoute } from "wouter-preact";
import { type SearchResult, searchSpotify } from "@/lib/music/app/search";
import type { Song } from "@/lib/music/model";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

type SearchType = "user" | "track" | "album" | "playlist" | "artist";

function stripUriPrefix(query: string, type: string): string {
	const prefix = `spotify:${type}:`;
	return query.startsWith(prefix) ? query.slice(prefix.length) : query;
}

const detailRoutes: Partial<Record<SearchType, (id: string) => string>> = {
	artist: (id) => `/artist/${id}`,
	album: (id) => `/album/${id}`,
	playlist: (id) => `/playlist/${id}`,
};

function SearchResultItem({ item }: { item: SearchResult }) {
	const coverUrl = item.coverUrl ?? "";
	const artists = (item.artists ?? []).join(", ");

	return (
		<Link href={`/playlist/${stripUriPrefix(item.uri, "playlist")}`}>
			<div class="rounded-md overflow-hidden cursor-pointer group">
				<div class="flex flex-col gap-2">
					<div class="aspect-square w-full overflow-hidden rounded-md">
						{coverUrl ? (
							<img
								src={coverUrl}
								alt={item.name}
								class="w-full h-full object-cover group-hover:opacity-80 transition-opacity"
							/>
						) : (
							<div class="h-full w-full bg-zinc-900" />
						)}
					</div>
					<h3 class="font-medium truncate text-sm text-zinc-300 group-hover:text-white transition-colors">
						{item.name}
					</h3>
					{artists && <p class="text-xs text-zinc-500 truncate">{artists}</p>}
					{item.ownerName && (
						<p class="text-xs text-zinc-500 truncate">By {item.ownerName}</p>
					)}
					{item.trackCount != null && (
						<p class="text-xs text-zinc-500 truncate">
							{item.trackCount} track{item.trackCount === 1 ? "" : "s"}
						</p>
					)}
				</div>
			</div>
		</Link>
	);
}

function resultToSong(item: SearchResult): Song {
	const id = stripUriPrefix(item.uri, "track");
	const coverUrl = item.coverUrl ?? "";
	return {
		id,
		name: item.name,
		duration: 0,
		share: {
			id,
			url: `https://open.spotify.com/track/${id}`,
		},
		album: {
			title: "",
			url: "",
			covers: coverUrl ? [coverUrl] : [],
		},
		artists: (item.artists ?? []).map((name) => ({ name })),
	};
}

export function SearchPage() {
	const [, params] = useRoute("/search/:type/:query");
	const type = params?.type as SearchType;
	const query = params?.query ? decodeURIComponent(params.query) : "";
	const [, setLocation] = useLocation();

	useMeta(
		`Jota | Search ${type}: ${query}`,
		`Search results for ${type} "${query}"`,
	);

	// artist/album/playlist search resolves to a single entity → redirect to its page
	const detailRoute = type ? detailRoutes[type] : undefined;
	const redirectId = detailRoute ? stripUriPrefix(query, type) : "";

	const { data, isLoading, isError } = useQuery({
		queryKey: ["spotify-search", type, query],
		queryFn: () => searchSpotify(type, query),
		enabled: !detailRoute && !!query,
	});

	useLayoutEffect(() => {
		if (detailRoute && redirectId) {
			setLocation(detailRoute(redirectId));
		}
	}, [detailRoute, redirectId]);

	if (detailRoute) {
		return <DefaultLayout class="gap-6 h-full" />;
	}

	if (isError) {
		return (
			<DefaultLayout class="gap-6 h-full">
				<div class="flex flex-col gap-6 h-full items-center justify-center">
					<p class="text-red-400">Failed to search</p>
				</div>
			</DefaultLayout>
		);
	}

	const results = (data as SearchResult[]) ?? [];

	if (type === "track") {
		const songs = results.map(resultToSong);
		return (
			<DefaultLayout class="gap-4 h-full">
				<div class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
					<header>
						<h2 class="text-xl font-semibold leading-tight">{query}</h2>
					</header>
					{isLoading ? (
						<div class="flex min-h-0 flex-1 items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
							<Loader size={24} class="animate-spin" />
						</div>
					) : songs.length === 0 ? (
						<p class="text-sm text-zinc-500">No results found.</p>
					) : (
						<Virtualization songs={songs} />
					)}
				</div>
			</DefaultLayout>
		);
	}

	return (
		<DefaultLayout class="gap-6 h-full">
			<div class="flex flex-col gap-6 h-full">
				<header>
					<h2 class="text-xl font-semibold leading-tight">
						Playlists by {query}
					</h2>
				</header>

				{isLoading ? (
					<div class="flex items-center gap-2 text-sm text-zinc-500">
						<Loader size={14} class="animate-spin" />
						Loading...
					</div>
				) : results.length === 0 ? (
					<p class="text-sm text-zinc-500">No playlists found.</p>
				) : (
					<div class="grid grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
						{results.map((item) => (
							<SearchResultItem key={item.uri} item={item} />
						))}
					</div>
				)}
			</div>
		</DefaultLayout>
	);
}

export default SearchPage;
