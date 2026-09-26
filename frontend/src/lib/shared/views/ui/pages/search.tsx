import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { Loader, Pause, Play } from "lucide-preact";
import { useLayoutEffect, useRef } from "preact/hooks";
import { Link, useLocation, useParams } from "wouter-preact";
import {
	type SearchResult,
	searchResultToSong,
	searchSpotify,
	stripUriPrefix,
} from "@/lib/music/app/search";
import { playAlbum, playPlaylist } from "@/lib/music/views/play";
import { isPlaying } from "@/lib/music/views/stores/audio";
import { queueSource } from "@/lib/music/views/stores/queue";
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { UserHeader } from "@/lib/music/views/ui/user/header";
import { t } from "@/lib/shared/i18n";
import { cover } from "@/lib/shared/utils/cover";
import {
	type Action,
	openContextMenu,
} from "@/lib/shared/views/ui/components/context-menu";
import { Image } from "@/lib/shared/views/ui/components/image";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

type SearchType = "user" | "track" | "album" | "playlist" | "artist";

const detailRoutes: Partial<Record<SearchType, (id: string) => string>> = {
	artist: (id) => `/artist/${id}`,
	album: (id) => `/album/${id}`,
	playlist: (id) => `/playlist/${id}`,
};

function SearchResultItem({ item }: { item: SearchResult }) {
	const queryClient = useQueryClient();
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
		} else if (isAlbum) {
			actions.push({
				...play,
				run: () =>
					void playAlbum(queryClient, stripUriPrefix(item.uri, "album")),
			});
		}
		if (actions.length > 0) openContextMenu(actions, e);
	}

	return (
		<Link href={`/playlist/${playlistId}`} onContextMenu={openMenu}>
			<div class="group cursor-pointer overflow-hidden rounded-md">
				<div class="flex flex-col gap-2">
					<div class="relative aspect-square w-full overflow-hidden rounded-md">
						{coverUrl ? (
							<Image
								src={cover(coverUrl, 320)}
								alt={item.name}
								class="h-full w-full object-cover transition-opacity group-hover:opacity-80"
							/>
						) : (
							<div class="h-full w-full bg-zinc-900" />
						)}
						{isPlaylist && (
							<div class="absolute right-1 top-1 flex gap-1 opacity-0 transition group-hover:opacity-100 group-focus-within:opacity-100">
								<PlaylistPlayButton id={playlistId} />
							</div>
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
		gcTime: 0,
	});

	useLayoutEffect(() => {
		if (detailRoute && redirectId) {
			setLocation(detailRoute(redirectId), { replace: true });
		}
	}, [detailRoute, redirectId, setLocation]);

	if (isError) {
		return (
			<DefaultLayout class="gap-6">
				<p class="text-red-400">{t("pages.search.failed")}</p>
			</DefaultLayout>
		);
	}

	const results = (data as SearchResult[]) ?? [];

	if (type === "track") {
		const songs = results.map(searchResultToSong);

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
						<p class="text-sm text-zinc-500">{t("common.noResults")}</p>
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
					<UserHeader source="spotify" identifier={query} />
				) : (
					<header class="shrink-0">
						<h2 class="text-xl font-semibold leading-tight">
							{t("pages.search.playlistsBy", { query })}
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
								{t("common.loading")}
							</div>
						) : results.length === 0 ? (
							<p class="text-sm text-zinc-500">
								{t("pages.search.noPlaylists")}
							</p>
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
