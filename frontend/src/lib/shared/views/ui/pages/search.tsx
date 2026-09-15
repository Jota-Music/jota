import { useQuery } from "@tanstack/preact-query";
import { Loader } from "lucide-preact";
import { useLayoutEffect, useRef } from "preact/hooks";
import { Link, useLocation } from "wouter-preact";
import { type SearchResult, searchSpotify } from "@/lib/music/app/search";
import type { Song } from "@/lib/music/model";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { UserHeader } from "@/lib/music/views/ui/user/header";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";
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
			<div class="group cursor-pointer overflow-hidden rounded-md">
				<div class="flex flex-col gap-2">
					<div class="aspect-square w-full overflow-hidden rounded-md">
						{coverUrl ? (
							<img
								src={coverUrl}
								alt={item.name}
								class="h-full w-full object-cover transition-opacity group-hover:opacity-80"
							/>
						) : (
							<div class="h-full w-full bg-zinc-900" />
						)}
					</div>

					<h3 class="truncate text-sm font-medium text-zinc-300 transition-colors group-hover:text-white">
						{item.name}
					</h3>

					{artists && <p class="truncate text-xs text-zinc-500">{artists}</p>}

					{item.ownerName && (
						<p class="truncate text-xs text-zinc-500">By {item.ownerName}</p>
					)}

					{item.trackCount != null && (
						<p class="truncate text-xs text-zinc-500">
							{item.trackCount} track
							{item.trackCount === 1 ? "" : "s"}
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
		url: "",
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

function getSearchParams(): {
	type: SearchType | undefined;
	query: string;
} {
	const pathname = window.location.pathname;
	const match = pathname.match(/^\/search\/([^/]+)\/(.+)$/);

	if (!match) {
		return {
			type: undefined,
			query: "",
		};
	}

	const [, rawType, rawQuery] = match;

	const validTypes: SearchType[] = [
		"user",
		"track",
		"album",
		"playlist",
		"artist",
	];

	const type = validTypes.includes(rawType as SearchType)
		? (rawType as SearchType)
		: undefined;

	return {
		type,
		query: rawQuery ? decodeURIComponent(rawQuery) : "",
	};
}

function PlaylistGrid({ results }: { results: SearchResult[] }) {
	return (
		<div class="grid grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
			{results.map((item) => (
				<SearchResultItem key={item.uri} item={item} />
			))}
		</div>
	);
}

export function SearchPage() {
	const { type, query } = getSearchParams();
	const [, setLocation] = useLocation();

	const detailRoute = type ? detailRoutes[type] : undefined;
	const redirectId = detailRoute && type ? stripUriPrefix(query, type) : "";
	const listRef = useRef<HTMLElement>(null);

	const { data, isLoading, isError } = useQuery({
		queryKey: ["spotify-search", type, query],
		queryFn: () => {
			if (!type) throw new Error("search type is required");
			return searchSpotify(type, query);
		},
		enabled: !detailRoute && !!type && !!query,
	});

	useLayoutEffect(() => {
		if (detailRoute && redirectId) {
			setLocation(detailRoute(redirectId), { replace: true });
		}
	}, [detailRoute, redirectId, setLocation]);

	if (isError) {
		return (
			<DefaultLayout class="gap-6">
				<p class="text-red-400">Failed to search</p>
			</DefaultLayout>
		);
	}

	const results = (data as SearchResult[]) ?? [];

	if (type === "track") {
		const songs = results.map(resultToSong);

		return (
			<DefaultLayout class="gap-4">
				<div class="flex flex-col gap-2 min-h-0 flex-1 pb-6">
					<header class="shrink-0">
						<h2 class="text-xl font-semibold leading-tight">{query}</h2>
					</header>

					{isLoading ? (
						<div class="flex items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
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
		<DefaultLayout class="gap-4">
			<div class="flex flex-col gap-4 min-h-0 flex-1">
				{type === "user" ? (
					<UserHeader username={query} />
				) : (
					<header class="shrink-0">
						<h2 class="text-xl font-semibold leading-tight">
							Playlists by {query}
						</h2>
					</header>
				)}

				<div class="relative min-h-0 flex-1">
					<section
						ref={listRef}
						class="h-full flex flex-col gap-3 overflow-y-auto pb-8"
					>
						{isLoading ? (
							<div class="flex items-center gap-2 text-sm text-zinc-500">
								<Loader size={14} class="animate-spin" />
								Loading...
							</div>
						) : results.length === 0 ? (
							<p class="text-sm text-zinc-500">No playlists found.</p>
						) : (
							<PlaylistGrid results={results} />
						)}
					</section>
					<Scrollbar target={listRef} />
				</div>
			</div>
		</DefaultLayout>
	);
}

export default SearchPage;
