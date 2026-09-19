import { useSignal } from "@preact/signals";
import { useQuery } from "@tanstack/preact-query";
import { CirclePlay, ListVideo, Loader, Pause, Play } from "lucide-preact";
import { useEffect, useRef } from "preact/hooks";
import { Link, useRoute } from "wouter-preact";
import {
	searchYouTube,
	searchYouTubePlaylists,
	youtubeVideoToSong,
} from "@/lib/music/app/search";
import { parseYoutubeLink } from "@/lib/music/app/youtube-link";
import type { PlaylistSummary, Song } from "@/lib/music/model";
import { playPlaylist } from "@/lib/music/views/play";
import { isPlaying } from "@/lib/music/views/stores/audio";
import { playFromQueueSelection } from "@/lib/music/views/stores/player";
import { queueSource } from "@/lib/music/views/stores/queue";
import { clearSelection } from "@/lib/music/views/stores/selection";
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import SelectionBar from "@/lib/music/views/ui/track/selection-bar";
import { YouTubeVideoRow } from "@/lib/music/views/ui/track/youtube-video-row";
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
	const picked = useSignal<{ query: string; tab: Tab } | null>(null);
	const listRef = useRef<HTMLElement>(null);

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
		? picked.value?.query === query
			? picked.value.tab
			: target
		: videosAvailable
			? "videos"
			: "playlists";

	useEffect(() => {
		clearSelection();
	}, [activeTab]);

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
								onClick={() => {
									picked.value = { query, tab: "videos" };
								}}
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
								onClick={() => {
									picked.value = { query, tab: "playlists" };
								}}
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
										<PlaylistCard key={playlist.id} playlist={playlist} />
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
								<YouTubeVideoRow
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

function PlaylistCard({ playlist }: { playlist: PlaylistSummary }) {
	const playing = queueSource.value === playlist.id && isPlaying.value;
	return (
		// biome-ignore lint/a11y/noStaticElementInteractions: right-click opens the playlist actions
		<div
			class="group flex flex-col gap-2"
			onContextMenu={(e) => {
				openContextMenu(
					[
						{
							icon: playing ? Pause : Play,
							label: playing ? t("music.pause") : t("music.play"),
							run: () => void playPlaylist(playlist.id),
						},
					],
					e,
				);
			}}
		>
			<div class="relative aspect-square w-full overflow-hidden rounded-md bg-zinc-900">
				<Link href={`/playlist/${playlist.id}`}>
					<PlaylistCover
						src={playlist.cover}
						alt={playlist.name}
						imgClass="h-full w-full object-cover transition-opacity group-hover:opacity-80"
					/>
				</Link>

				<div class="absolute right-2 top-2 flex gap-1 opacity-0 transition group-hover:opacity-100 group-focus-within:opacity-100">
					<PlaylistPlayButton id={playlist.id} />
				</div>
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
