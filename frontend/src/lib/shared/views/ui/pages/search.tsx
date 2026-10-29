import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { Pause, Play } from "lucide-preact";
import { useLayoutEffect, useRef } from "preact/hooks";
import { Link, useLocation, useParams } from "wouter-preact";
import {
	type SearchResult,
	searchResultToSong,
	searchSpotify,
	stripUriPrefix,
} from "@/lib/music/app/search";
import { pageLink } from "@/lib/music/app/share";
import { spotifyRef } from "@/lib/music/app/spotify-link";
import { playAlbum, playPlaylist } from "@/lib/music/views/play";
import { isPlaying } from "@/lib/music/views/stores/audio";
import { queueSource } from "@/lib/music/views/stores/queue";
import { hover } from "@/lib/music/views/ui/cover-hover";
import { SaveButton } from "@/lib/music/views/ui/follow";
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import { useSaved } from "@/lib/music/views/ui/playlist/saved";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { shareLink } from "@/lib/music/views/ui/share";
import { UserHeader } from "@/lib/music/views/ui/user/header";
import { t } from "@/lib/shared/i18n";
import { cover } from "@/lib/shared/utils/cover";
import { cn } from "@/lib/shared/utils/tw";
import {
	type Action,
	openContextMenu,
} from "@/lib/shared/views/ui/components/context-menu";
import { CoverFallback } from "@/lib/shared/views/ui/components/cover-fallback";
import { Image } from "@/lib/shared/views/ui/components/image";
import { Loading } from "@/lib/shared/views/ui/components/loading";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

type SearchType = "user" | "track" | "album" | "playlist" | "artist";

const DETAIL_ROUTES: Partial<Record<SearchType, (id: string) => string>> = {
	artist: (id) => `/artist/${id}`,
	album: (id) => `/album/${id}`,
	playlist: (id) => `/playlist/${id}`,
};

function SearchResultItem({ item }: { item: SearchResult }) {
	const queryClient = useQueryClient();
	const { toggle } = useSaved();
	const coverUrl = item.coverUrl ?? "";
	const artists = (item.artists ?? []).map((a) => a.name).join(", ");
	const isPlaylist = item.uri.startsWith("spotify:playlist:");
	const isAlbum = item.uri.startsWith("spotify:album:");
	const playlistId = stripUriPrefix(item.uri, "playlist");

	function openMenu(e: MouseEvent) {
		const actions: Action[] = [];
		const id = isPlaylist ? playlistId : stripUriPrefix(item.uri, "album");
		const playing = queueSource.value === id && isPlaying.value;
		const play = {
			icon: playing ? Pause : Play,
			label: playing ? t("music.pause") : t("music.play"),
		};
		if (isPlaylist) {
			actions.push({
				...play,
				run: () => void playPlaylist(queryClient, playlistId),
			});
			actions.push(toggle(playlistId));
		} else if (isAlbum) {
			actions.push({
				...play,
				run: () =>
					void playAlbum(queryClient, stripUriPrefix(item.uri, "album")),
			});
		}
		if (isPlaylist || isAlbum) {
			actions.push(
				...shareLink(pageLink(isPlaylist ? "playlist" : "album", id)),
			);
		}
		if (actions.length > 0) openContextMenu(actions, e);
	}

	return (
		<Link href={`/playlist/${playlistId}`} onContextMenu={openMenu}>
			<div class="group cursor-pointer overflow-hidden rounded-md">
				<div class="flex flex-col gap-2">
					<div class="relative aspect-square w-full overflow-hidden rounded-md">
						<Image
							src={cover(coverUrl, 320)}
							alt={item.name}
							class="h-full w-full object-cover transition-opacity group-hover:opacity-80"
							fallback={<CoverFallback />}
						/>
						{isPlaylist && (
							<>
								<div class={cn(hover, "left-1 top-1")}>
									<SaveButton id={playlistId} />
								</div>
								<div class={cn(hover, "right-1 top-1")}>
									<PlaylistPlayButton id={playlistId} />
								</div>
							</>
						)}
					</div>

					<h3 class="truncate text-sm font-medium text-zinc-300 transition-colors group-hover:text-white">
						{item.name}
					</h3>

					{artists && <p class="truncate text-xs text-zinc-500">{artists}</p>}

					{item.ownerName && (
						<p class="truncate text-xs text-zinc-500">{item.ownerName}</p>
					)}

					{item.trackCount != null && (
						<p class="truncate text-xs text-zinc-500">
							{t("music.trackCount", { count: item.trackCount })}
						</p>
					)}
				</div>
			</div>
		</Link>
	);
}

function PlaylistGrid({ results }: { results: SearchResult[] }) {
	return (
		<div class="grid grid-cols-3 gap-0.5 md:gap-4">
			{results.map((item) => (
				<SearchResultItem key={item.uri} item={item} />
			))}
		</div>
	);
}

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

	// Spotify only resolves artists, albums and playlists by reference, so a
	// bare name is not something the detail page can open.
	const detailRoute = type ? DETAIL_ROUTES[type] : undefined;
	const ref = spotifyRef(type, query);
	const redirectId = detailRoute && ref && ref.type === type ? ref.id : "";
	const listRef = useRef<HTMLElement>(null);

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
			<p class="text-sm text-zinc-500">
				{t("pages.search.referenceOnly", { type: t(`search.type.${type}`) })}
			</p>
		);
	}

	if (isError) {
		return <p class="text-red-400">{t("pages.search.failed")}</p>;
	}

	const results = (data as SearchResult[]) ?? [];

	if (type === "track") {
		const songs = results.map(searchResultToSong);

		return (
			<div class="flex flex-col gap-2 min-h-0 flex-1 pb-6">
				<header class="shrink-0">
					<h2 class="line-clamp-2 break-words text-xl font-semibold leading-tight">
						{query}
					</h2>
				</header>

				{isLoading ? (
					<Loading />
				) : songs.length === 0 ? (
					<p class="text-sm text-zinc-500">{t("common.noResults")}</p>
				) : (
					<Virtualization songs={songs} />
				)}
			</div>
		);
	}

	return (
		<div class="flex flex-col gap-4 min-h-0 flex-1">
			{type === "user" ? (
				<UserHeader source="spotify" identifier={query} />
			) : (
				<header class="shrink-0">
					<h2 class="line-clamp-2 break-words text-xl font-semibold leading-tight">
						{t("pages.search.playlistsBy", { query })}
					</h2>
				</header>
			)}

			<div class="relative min-h-0 flex-1">
				<section
					ref={listRef}
					class="h-full flex flex-col gap-3 overflow-y-auto pr-[15px] pb-8"
				>
					{isLoading ? (
						<Loading />
					) : results.length === 0 ? (
						<p class="text-sm text-zinc-500">{t("pages.search.noPlaylists")}</p>
					) : (
						<PlaylistGrid results={results} />
					)}
				</section>
				<Scrollbar target={listRef} />
			</div>
		</div>
	);
}
