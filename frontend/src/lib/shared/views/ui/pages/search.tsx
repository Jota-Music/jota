import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { useLayoutEffect } from "preact/hooks";
import { useLocation, useParams } from "wouter-preact";
import {
	type SearchResult,
	searchResultToSong,
	searchSpotify,
	stripUriPrefix,
} from "@/lib/music/app/search";
import { spotifyRef } from "@/lib/music/app/spotify-link";
import { playAlbum, playPlaylist } from "@/lib/music/views/play";
import { SaveButton } from "@/lib/music/views/ui/follow";
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import { useSaved } from "@/lib/music/views/ui/playlist/saved";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { type Item, Shelf } from "@/lib/music/views/ui/shelf";
import { UserHeader } from "@/lib/music/views/ui/user/header";
import { t } from "@/lib/shared/i18n";
import { Rows } from "@/lib/shared/views/ui/components/loading";

type SearchType = "user" | "track" | "album" | "playlist" | "artist";

const DETAIL_ROUTES: Partial<Record<SearchType, (id: string) => string>> = {
	artist: (id) => `/artist/${id}`,
	album: (id) => `/album/${id}`,
	playlist: (id) => `/playlist/${id}`,
};

export function SearchPage() {
	const params = useParams<{ type: string; query: string }>();
	const validTypes: SearchType[] = [
		"user",
		"track",
		"album",
		"playlist",
		"artist",
	];
	const type = validTypes.includes(params.type as SearchType)
		? (params.type as SearchType)
		: undefined;
	const query = params.query ? decodeURIComponent(params.query) : "";
	const [, setLocation] = useLocation();
	const queryClient = useQueryClient();
	const { toggle } = useSaved();

	// Spotify only resolves artists, albums and playlists by reference, so a
	// bare name is not something the detail page can open.
	const detailRoute = type ? DETAIL_ROUTES[type] : undefined;
	const ref = spotifyRef(type, query);
	const redirectId = detailRoute && ref && ref.type === type ? ref.id : "";

	const { data, isLoading, isError } = useQuery({
		queryKey: ["spotify-search", type, query],
		queryFn: () => {
			if (!type) throw new Error("search type is required");
			return searchSpotify(type, query);
		},
		enabled: !detailRoute && !!type && !!query,
		gcTime: 0,
	});

	useLayoutEffect(() => {
		if (detailRoute && redirectId) {
			setLocation(detailRoute(redirectId), { replace: true });
		}
	}, [detailRoute, redirectId, setLocation]);

	if (detailRoute && !redirectId) {
		return (
			<p class="px-3 pt-2 text-sm text-zinc-500 md:px-0 md:pt-0">
				{t("pages.search.referenceOnly", { type: t(`search.type.${type}`) })}
			</p>
		);
	}

	if (isError) {
		return (
			<p class="px-3 pt-2 text-red-400 md:px-0 md:pt-0">
				{t("pages.search.failed")}
			</p>
		);
	}

	const results = (data as SearchResult[]) ?? [];
	const tracks = type === "track";
	// A profile search answers with that user's playlists.
	const playlists = type === "playlist" || type === "user";

	const songs = tracks ? results.map(searchResultToSong) : [];

	// The shelf names the search itself: a result page has no header of its own,
	// so the query rides in the bar above the cards, with the categories that the
	// header's search box picked before the results existed.
	const items: Item[] = tracks
		? []
		: results.map((item) => {
				const id = stripUriPrefix(item.uri, type ?? "");
				const subtitle = [
					(item.artists ?? []).map((artist) => artist.name).join(", "),
					item.ownerName,
					item.trackCount != null
						? t("music.trackCount", { count: item.trackCount })
						: "",
				]
					.filter(Boolean)
					.join(" • ");

				return {
					id,
					name: item.name,
					cover: item.coverUrl,
					subtitle: subtitle || undefined,
					menu: playlists ? [toggle(id)] : [],
				};
			});

	const to = (id: string) =>
		type === "album"
			? `/album/${id}`
			: type === "artist"
				? `/artist/${id}`
				: `/playlist/${id}`;

	const onPlay =
		type === "album"
			? (id: string) => void playAlbum(queryClient, id)
			: playlists
				? (id: string) => void playPlaylist(queryClient, id)
				: undefined;

	return (
		<div class="flex flex-col gap-4 min-h-0 flex-1 overflow-hidden">
			{type === "user" && <UserHeader source="spotify" identifier={query} />}

			{tracks ? (
				// Rows are not cards: the list keeps its own panel and scroll, and the
				// query rides in the header the shelf would otherwise carry.
				<div class="flex min-h-0 flex-1 flex-col gap-2">
					<h2 class="line-clamp-2 shrink-0 break-words text-xl font-semibold leading-tight px-3 pt-2 md:px-0 md:pt-0">
						{query}
					</h2>
					{isLoading ? (
						<Rows />
					) : songs.length === 0 ? (
						<p class="text-sm text-zinc-500">{t("common.noResults")}</p>
					) : (
						<Virtualization songs={songs} />
					)}
				</div>
			) : (
				<Shelf
					items={items}
					to={to}
					viewKey="search_results_view"
					title={query}
					isLoading={isLoading}
					emptyMessage={t(
						playlists ? "pages.search.noPlaylists" : "common.noResults",
					)}
					onPlay={onPlay}
					leading={playlists ? (id) => <SaveButton id={id} /> : undefined}
					actions={
						playlists ? (id) => <PlaylistPlayButton id={id} /> : undefined
					}
				/>
			)}
		</div>
	);
}
