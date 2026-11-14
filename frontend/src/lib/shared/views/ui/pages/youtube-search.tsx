import { useSignal } from "@preact/signals";
import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { CirclePlay, ListVideo, User } from "lucide-preact";
import { useEffect, useRef } from "preact/hooks";
import { useRoute } from "wouter-preact";
import {
	searchYouTube,
	searchYouTubeChannels,
	searchYouTubePlaylists,
	youtubeVideoToSong,
} from "@/lib/music/app/search";
import { parseYoutubeLink } from "@/lib/music/app/youtube-link";
import type { Song } from "@/lib/music/model";
import { playPlaylist } from "@/lib/music/views/play";
import { playFromQueueSelection } from "@/lib/music/views/stores/player";
import { clearSelection } from "@/lib/music/views/stores/selection";
import { SaveButton } from "@/lib/music/views/ui/follow";
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import { useSaved } from "@/lib/music/views/ui/playlist/saved";
import { Shelf } from "@/lib/music/views/ui/shelf";
import { YouTubeVideoRow } from "@/lib/music/views/ui/track/youtube-video-row";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { Rows } from "@/lib/shared/views/ui/components/loading";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

type Tab = "videos" | "playlists" | "channels";

const TAB_ICONS = {
	videos: CirclePlay,
	playlists: ListVideo,
	channels: User,
} as const;

const TAB_TITLES = {
	videos: "pages.youtube.videos",
	playlists: "pages.youtube.playlists",
	channels: "pages.youtube.channels",
} as const;

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
	const queryClient = useQueryClient();
	const { toggle } = useSaved();

	const target = queryTarget(query);
	const direct = isDirectQuery(query);
	const picked = useSignal<{ query: string; tab: Tab } | null>(null);
	const listRef = useRef<HTMLElement>(null);

	const wantVideos = !!query && (!direct || target === "videos");
	const wantPlaylists = !!query && (!direct || target === "playlists");
	// A pasted link is never a channel, so it only needs the two other queries.
	const wantChannels = !!query && !direct;

	const videosQuery = useQuery({
		queryKey: ["youtube-search", query],
		queryFn: () => searchYouTube(query),
		enabled: wantVideos,
		gcTime: 0,
	});

	const playlistsQuery = useQuery({
		queryKey: ["youtube-playlist-search", query],
		queryFn: () => searchYouTubePlaylists(query),
		enabled: wantPlaylists,
		gcTime: 0,
	});

	const channelsQuery = useQuery({
		queryKey: ["youtube-channel-search", query],
		queryFn: () => searchYouTubeChannels(query),
		enabled: wantChannels,
		gcTime: 0,
	});

	const videos = videosQuery.data ?? [];
	const playlists = playlistsQuery.data ?? [];
	const channels = channelsQuery.data ?? [];

	// A category with no results drops out of the switcher, but one that is
	// still loading stays in so the tabs do not jump around as answers land.
	const tabs = (
		[
			wantVideos &&
				(videosQuery.isSuccess ? videos.length > 0 : true) &&
				"videos",
			wantPlaylists &&
				(playlistsQuery.isSuccess ? playlists.length > 0 : true) &&
				"playlists",
			wantChannels &&
				(channelsQuery.isSuccess ? channels.length > 0 : true) &&
				"channels",
		] as const
	).filter((tab) => tab !== false);

	const pickedTab = picked.value?.query === query ? picked.value.tab : target;
	const activeTab: Tab = tabs.includes(pickedTab)
		? pickedTab
		: (tabs[0] ?? "videos");

	useEffect(() => {
		clearSelection();
	}, [activeTab]);

	const songs: Song[] = videos.map(youtubeVideoToSong);

	function handleClick(song: Song) {
		void playFromQueueSelection(songs, song);
	}

	const loading =
		(wantVideos && videosQuery.isLoading) ||
		(wantPlaylists && playlistsQuery.isLoading) ||
		(wantChannels && channelsQuery.isLoading);
	// A category with no results drops out of the switcher, so an empty answer
	// leaves nothing to switch between.
	if (tabs.length === 0 && !loading) {
		return <p class="text-sm text-zinc-500">{t("pages.youtube.noResults")}</p>;
	}

	// Videos are rows of their own; playlists and channels are items of the
	// shelf, so only those two offer the grid/compact switch.
	const listable = activeTab !== "videos";
	const activeLoading =
		activeTab === "channels"
			? channelsQuery.isLoading
			: playlistsQuery.isLoading;

	const items =
		activeTab === "channels"
			? channels.map((channel) => ({
					id: channel.id,
					name: channel.name,
					cover: channel.avatar,
				}))
			: playlists.map((playlist) => ({
					id: playlist.id,
					name: playlist.name,
					cover: playlist.cover,
					subtitle: playlist.subtitle,
					menu: [toggle(playlist.id)],
				}));

	const emptyMessage = !listable
		? t("pages.youtube.noResults")
		: activeTab === "channels"
			? t(
					channelsQuery.isError
						? "pages.youtube.failed"
						: "pages.youtube.noChannels",
				)
			: t(
					playlistsQuery.isError
						? "pages.youtube.failed"
						: "pages.youtube.noPlaylists",
				);

	const videoBody = videosQuery.isLoading ? (
		<Rows />
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
				class="h-full overflow-y-auto pl-1 pr-[15px]"
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
	);

	return (
		<Shelf
			items={listable ? items : []}
			to={(id) =>
				activeTab === "channels" ? `/youtube/user/${id}` : `/playlist/${id}`
			}
			viewKey="youtube_search_view"
			title={query}
			toolbar={
				tabs.length > 1 && (
					<div class="flex items-center gap-1 rounded-lg border border-zinc-800 bg-zinc-950 p-1">
						{tabs.map((tab) => {
							const Icon = TAB_ICONS[tab];
							return (
								<button
									key={tab}
									type="button"
									title={t(TAB_TITLES[tab])}
									onClick={() => {
										picked.value = { query, tab };
									}}
									class={cn(
										"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md transition-colors",
										activeTab === tab
											? "bg-zinc-800 text-white"
											: "text-zinc-500 hover:text-zinc-300",
									)}
								>
									<Icon size={16} />
								</button>
							);
						})}
					</div>
				)
			}
			modes={listable}
			isLoading={listable && activeLoading}
			emptyMessage={emptyMessage}
			onPlay={
				activeTab === "playlists"
					? (id) => void playPlaylist(queryClient, id)
					: undefined
			}
			leading={
				activeTab === "playlists" ? (id) => <SaveButton id={id} /> : undefined
			}
			actions={
				activeTab === "playlists"
					? (id) => <PlaylistPlayButton id={id} />
					: undefined
			}
		>
			{listable ? undefined : videoBody}
		</Shelf>
	);
}
