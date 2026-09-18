import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { CirclePlay, Heart, ListVideo, Loader } from "lucide-preact";
import { useEffect, useRef, useState } from "preact/hooks";
import { Link, useRoute } from "wouter-preact";
import {
	searchYouTube,
	searchYouTubePlaylists,
	youtubeVideoToSong,
} from "@/lib/music/app/search";
import { parseYoutubeLink } from "@/lib/music/app/youtube-link";
import {
	addYouTubePlaylist,
	getYouTubePlaylists,
	removeYouTubePlaylist,
} from "@/lib/music/app/youtube-playlist";
import type { PlaylistSummary, Song } from "@/lib/music/model";
import { rowSelect } from "@/lib/music/views/hooks/row-select";
import { playFromQueueSelection } from "@/lib/music/views/stores/player";
import {
	clearSelection,
	selectedSongs,
	toggleSelection,
} from "@/lib/music/views/stores/selection";
import SelectionBar from "@/lib/music/views/ui/components/selection-bar";
import TrackActions, {
	buildActions,
} from "@/lib/music/views/ui/components/track-actions";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { openContextMenu } from "@/lib/shared/views/ui/components/context-menu";
import PlaylistCover from "@/lib/shared/views/ui/components/playlist-cover";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

type Tab = "videos" | "playlists";

// A pasted YouTube link/ID decides which category it points to.
function queryTarget(query: string): Tab {
	return parseYoutubeLink(query)?.type === "playlist" ? "playlists" : "videos";
}

// A direct query is a YouTube URL or a bare video/playlist ID: it resolves to a
// single known category, so there is nothing to switch between.
function isDirectQuery(query: string): boolean {
	const trimmed = query.trim();
	if (parseYoutubeLink(trimmed)) return true;
	try {
		const protocol = new URL(trimmed).protocol;
		return protocol === "http:" || protocol === "https:";
	} catch {
		return false;
	}
}

export function YouTubeSearchPage() {
	const [, params] = useRoute<{ query: string }>("/search/youtube/:query");

	const query = params?.query ? decodeURIComponent(params.query) : "";

	const target = queryTarget(query);
	const direct = isDirectQuery(query);
	const [tab, setTab] = useState<Tab>(target);
	const queryClient = useQueryClient();
	const listRef = useRef<HTMLElement>(null);

	useEffect(() => {
		setTab(target);
	}, [target]);

	const wantVideos = !!query && (!direct || target === "videos");
	const wantPlaylists = !!query && (!direct || target === "playlists");

	const videosQuery = useQuery({
		queryKey: ["youtube-search", query],
		queryFn: () => searchYouTube(query),
		enabled: wantVideos,
	});

	const playlistsQuery = useQuery({
		queryKey: ["youtube-playlist-search", query],
		queryFn: () => searchYouTubePlaylists(query),
		enabled: wantPlaylists,
	});

	const videos = videosQuery.data ?? [];
	const playlists = playlistsQuery.data ?? [];

	const videosAvailable =
		wantVideos && (videosQuery.isSuccess ? videos.length > 0 : true);
	const playlistsAvailable =
		wantPlaylists && (playlistsQuery.isSuccess ? playlists.length > 0 : true);

	// Only offer the switcher when both categories actually have content.
	const bothAvailable = videosAvailable && playlistsAvailable;
	const activeTab: Tab = bothAvailable
		? tab
		: videosAvailable
			? "videos"
			: "playlists";

	useEffect(() => {
		clearSelection();
	}, [activeTab]);

	const savedQuery = useQuery({
		queryKey: ["youtube-playlists"],
		queryFn: getYouTubePlaylists,
		enabled: activeTab === "playlists",
	});

	const saved = new Set((savedQuery.data ?? []).map((p) => p.id));
	const [overrides, setOverrides] = useState<Record<string, boolean>>({});

	function isSaved(id: string): boolean {
		return overrides[id] ?? saved.has(id);
	}

	const toggle = useMutation({
		mutationFn: async ({ id, next }: { id: string; next: boolean }) => {
			if (next) {
				await addYouTubePlaylist(id);
			} else {
				await removeYouTubePlaylist(id);
			}
		},
		onMutate: ({ id, next }) => {
			setOverrides((o) => ({ ...o, [id]: next }));
			return { id, prev: !next };
		},
		onError: (_error, _vars, ctx) => {
			if (ctx) setOverrides((o) => ({ ...o, [ctx.id]: ctx.prev }));
		},
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["youtube-playlists"] }),
	});

	const songs: Song[] = videos.map(youtubeVideoToSong);

	function handleClick(song: Song) {
		void playFromQueueSelection(songs, song);
	}

	const loading =
		(wantVideos && videosQuery.isLoading) ||
		(wantPlaylists && playlistsQuery.isLoading);
	const hasResults = videosAvailable || playlistsAvailable;

	return (
		<DefaultLayout class="gap-4">
			<div class="flex flex-col gap-2 min-h-0 flex-1 pb-6">
				<header class="flex shrink-0 flex-col gap-3">
					<h2 class="text-xl font-semibold leading-tight">{query}</h2>
					{bothAvailable && (
						<div class="flex items-center gap-1 self-start rounded-lg border border-zinc-800 bg-zinc-950 p-1">
							<button
								type="button"
								title={t("pages.youtube.videos")}
								onClick={() => setTab("videos")}
								class={cn(
									"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md transition-colors",
									activeTab === "videos"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-zinc-300",
								)}
							>
								<CirclePlay size={18} />
							</button>
							<button
								type="button"
								title={t("pages.youtube.playlists")}
								onClick={() => setTab("playlists")}
								class={cn(
									"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md transition-colors",
									activeTab === "playlists"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-zinc-300",
								)}
							>
								<ListVideo size={16} />
							</button>
						</div>
					)}
				</header>

				<SelectionBar />

				{!hasResults && !loading ? (
					<p class="text-sm text-zinc-500">{t("pages.youtube.noResults")}</p>
				) : activeTab === "playlists" ? (
					<div class="relative min-h-0 flex-1">
						<section ref={listRef} class="h-full flex flex-col overflow-y-auto">
							{playlistsQuery.isLoading ? (
								<div class="flex items-center gap-2 text-sm text-zinc-500">
									<Loader size={14} class="animate-spin" />
									{t("common.loading")}
								</div>
							) : playlistsQuery.isError ? (
								<p class="text-red-400">{t("pages.youtube.failed")}</p>
							) : playlists.length === 0 ? (
								<p class="text-sm text-zinc-500">
									{t("pages.youtube.noPlaylists")}
								</p>
							) : (
								<div class="grid grid-cols-2 gap-3 pt-1 md:grid-cols-3 lg:grid-cols-4 md:gap-4">
									{playlists.map((playlist) => (
										<PlaylistCard
											key={playlist.id}
											playlist={playlist}
											saved={isSaved(playlist.id)}
											onToggle={() =>
												toggle.mutate({
													id: playlist.id,
													next: !isSaved(playlist.id),
												})
											}
										/>
									))}
								</div>
							)}
						</section>
						<Scrollbar target={listRef} />
					</div>
				) : videosQuery.isLoading ? (
					<div class="flex items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
						<Loader size={24} class="animate-spin" />
					</div>
				) : videosQuery.isError ? (
					<p class="text-red-400">{t("pages.youtube.failed")}</p>
				) : songs.length === 0 ? (
					<p class="text-sm text-zinc-500">{t("pages.youtube.noResults")}</p>
				) : (
					<div class="relative min-h-0 flex-1">
						{/* biome-ignore lint/a11y/noStaticElementInteractions: empty area clears the selection */}
						{/* biome-ignore lint/a11y/useKeyWithClickEvents: empty area clears the selection */}
						<section
							ref={listRef}
							class="h-full flex flex-col overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950"
							onClick={(e) => {
								const target = e.target as Element;
								if (!target.closest("[data-row]")) clearSelection();
							}}
						>
							{songs.map((song, index) => (
								<YouTubeVideoItem
									key={song.id}
									song={song}
									songs={songs}
									index={index}
									onClick={() => handleClick(song)}
								/>
							))}
						</section>
						<Scrollbar target={listRef} />
					</div>
				)}
			</div>
		</DefaultLayout>
	);
}

function PlaylistCard({
	playlist,
	saved,
	onToggle,
}: {
	playlist: PlaylistSummary;
	saved: boolean;
	onToggle: () => void;
}) {
	return (
		<div class="group flex flex-col gap-2">
			<div class="relative aspect-square w-full overflow-hidden rounded-md bg-zinc-900">
				<Link href={`/playlist/${playlist.id}`}>
					<PlaylistCover
						src={playlist.cover}
						alt={playlist.name}
						imgClass="h-full w-full object-cover transition-opacity group-hover:opacity-80"
					/>
				</Link>

				<button
					type="button"
					title={
						saved ? t("pages.youtube.removeHome") : t("pages.youtube.addHome")
					}
					onClick={(e) => {
						e.preventDefault();
						e.stopPropagation();
						onToggle();
					}}
					class={cn(
						"absolute right-2 top-2 flex h-8 w-8 cursor-pointer items-center justify-center rounded-full bg-black/70 transition hover:bg-black/90",
						saved ? "text-red-400" : "text-zinc-300 hover:text-white",
					)}
				>
					<Heart size={16} class={saved ? "fill-current" : ""} />
				</button>
			</div>

			<Link href={`/playlist/${playlist.id}`}>
				<div class="flex flex-col">
					<h3 class="truncate text-sm font-medium text-zinc-300 transition-colors group-hover:text-white">
						{playlist.name}
					</h3>
					{playlist.subtitle && (
						<p class="truncate text-xs text-zinc-500">{playlist.subtitle}</p>
					)}
				</div>
			</Link>
		</div>
	);
}

function YouTubeVideoItem({
	song,
	songs,
	index,
	onClick,
}: {
	song: Song;
	songs: Song[];
	index: number;
	onClick: () => void;
}) {
	const selected = selectedSongs.value.some((s) => s.id === song.id);

	const { onClick: onRowClick } = rowSelect({
		song,
		songs,
		index,
		onActivate: onClick,
	});

	return (
		<div
			data-row={song.id}
			role="none"
			class={cn(
				"flex w-full select-none items-center gap-3 border-b border-zinc-900 px-3 py-2 transition hover:bg-zinc-800/60",
				selected && "bg-zinc-800/60",
			)}
			onContextMenu={(e) => {
				const { actions } = buildActions({
					songs: [song],
					onToggleSelect: toggleSelection,
				});
				openContextMenu(actions, e);
			}}
		>
			<button
				type="button"
				onClick={onRowClick}
				class="flex min-w-0 flex-1 cursor-pointer items-center gap-3 text-left"
			>
				<PlaylistCover
					src={song.album?.covers?.[0]}
					alt={song.name}
					imgClass="size-10 shrink-0 rounded-md object-cover"
				/>

				<div class="min-w-0 flex-1">
					<p class="wrap-break-word text-balance text-sm text-white leading-snug">
						{song.name}
					</p>
				</div>
			</button>

			<TrackActions
				songs={[song]}
				selection={selected}
				onToggleSelect={toggleSelection}
			/>
		</div>
	);
}
