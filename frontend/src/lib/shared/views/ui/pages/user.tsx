import { useSignal } from "@preact/signals";
import { useQuery, useQueryClient } from "@tanstack/preact-query";
import {
	ArrowUpDown,
	CirclePlay,
	ListVideo,
	RefreshCw,
	Search,
	X,
} from "lucide-preact";
import { useEffect } from "preact/hooks";
import { useParams } from "wouter-preact";
import { getChannelInfo, getChannelVideos } from "@/lib/music/app/get-channel";
import {
	getUserPlaylists,
	revalidateUserPlaylists,
	type UserSource,
} from "@/lib/music/app/get-user-playlists";
import { isLiked, likedCover } from "@/lib/music/app/liked";
import type { PlaylistSummary, Song } from "@/lib/music/model";
import { playPlaylist } from "@/lib/music/views/play";
import { clearSelection } from "@/lib/music/views/stores/selection";
import { SaveButton } from "@/lib/music/views/ui/follow";
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import { followable, useSaved } from "@/lib/music/views/ui/playlist/saved";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { type Item, Shelf, Tabs } from "@/lib/music/views/ui/shelf";
import { UserHeader } from "@/lib/music/views/ui/user/header";
import { t } from "@/lib/shared/i18n";
import { Rows } from "@/lib/shared/views/ui/components/loading";

type Tab = "playlists" | "videos";

type OrderType = "added" | "reverse" | "duration-asc" | "duration-desc";

function sortVideos(videos: Song[], order: OrderType): Song[] {
	switch (order) {
		case "reverse":
			return [...videos].reverse();
		case "duration-asc":
			return [...videos].sort((a, b) => a.duration - b.duration);
		case "duration-desc":
			return [...videos].sort((a, b) => b.duration - a.duration);
		default:
			return videos;
	}
}

function toItem(playlist: PlaylistSummary): Item {
	return {
		id: playlist.id,
		name: isLiked(playlist.id) ? t("music.likedSongs") : playlist.name,
		cover: isLiked(playlist.id) ? likedCover : playlist.cover,
		subtitle: playlist.owner || playlist.subtitle,
	};
}

export function UserPage() {
	const params = useParams<{ username?: string; channelId?: string }>();
	const source: UserSource = params.channelId ? "youtube" : "spotify";
	const id = decodeURIComponent(params.channelId ?? params.username ?? "");
	const youtube = source === "youtube";

	const queryClient = useQueryClient();
	const { toggle } = useSaved();
	// A channel opens on its videos, which is also the tab that leads the
	// switcher: the highlighted tab and the first one have to agree.
	const activeTab = useSignal<Tab>("videos");
	const videosSearch = useSignal("");
	const videoOrder = useSignal<OrderType>("added");

	useEffect(() => {
		videosSearch.value = "";
		videoOrder.value = "added";
	}, [id]);

	const playlistsQuery = useQuery({
		queryKey: ["user-playlists", source, id],
		queryFn: () => getUserPlaylists(source, id),
		enabled: !!id,
	});

	const videosQuery = useQuery({
		queryKey: ["user-videos", id],
		queryFn: () => getChannelVideos(id),
		enabled: youtube,
	});

	const infoQuery = useQuery({
		queryKey: ["channel-info", id],
		queryFn: () => getChannelInfo(id),
		enabled: youtube,
	});

	async function handleRefresh() {
		if (id) {
			// Revalidate clears the playlist, video and info caches.
			await revalidateUserPlaylists(source, id).catch(() => {});
		}
		await queryClient.invalidateQueries({
			queryKey: ["user-playlists", source, id],
		});
		await queryClient.invalidateQueries({ queryKey: ["user-videos", id] });
		await queryClient.invalidateQueries({ queryKey: ["channel-info", id] });
	}

	const watchingVideos = youtube && activeTab.value === "videos";
	// Only the tab on screen has to answer: a channel whose playlists failed
	// still lists its videos, and the other way round.
	const failed = watchingVideos ? videosQuery.isError : playlistsQuery.isError;

	useEffect(() => {
		clearSelection();
	}, [activeTab.value]);

	if (failed) {
		return (
			<div class="flex flex-col items-start gap-3 p-6 text-sm">
				<p class="text-balance text-red-400">
					{t("pages.user.failed", { user: id })}
				</p>
				<button
					type="button"
					onClick={handleRefresh}
					title={t("common.retry")}
					class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white transition-colors"
				>
					<RefreshCw size={14} />
				</button>
			</div>
		);
	}

	const playlists = playlistsQuery.data ?? [];
	// Liked Songs still shows up, it just has nothing to follow.
	const items: Item[] = playlists.map(
		(p): Item => ({
			...toItem(p),
			menu: followable(p.id) ? [toggle(p.id)] : [],
		}),
	);
	const videos = videosQuery.data ?? [];

	const query = videosSearch.value.trim().toLowerCase();
	const base =
		query.length === 0
			? videos
			: videos.filter((video) => video.name.toLowerCase().includes(query));
	const filteredVideos = sortVideos(base, videoOrder.value);

	// The filters belong to the tab, not to the page: they only make sense over
	// the videos and disappear with them.
	const videoBody = (
		<div class="flex min-h-0 flex-1 flex-col gap-4">
			<div class="grid grid-cols-[1fr_auto] gap-2 px-3 md:px-0">
				<div class="relative flex-1">
					<Search
						size={16}
						class="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500"
					/>
					<input
						type="text"
						placeholder={t("music.playlist.searchPlaceholder")}
						value={videosSearch.value}
						onInput={(e) => {
							videosSearch.value = (e.target as HTMLInputElement).value;
						}}
						class="h-10 w-full rounded-md border border-zinc-800 bg-zinc-950 pl-9 pr-9 text-sm text-white outline-none focus:border-zinc-600"
					/>
					{videosSearch.value && (
						<button
							type="button"
							onClick={() => {
								videosSearch.value = "";
							}}
							class="absolute right-2 top-1/2 -translate-y-1/2 flex h-5 w-5 cursor-pointer items-center justify-center rounded text-zinc-500 hover:text-white hover:bg-zinc-800"
							aria-label={t("music.playlist.clearSearch")}
						>
							<X size={14} />
						</button>
					)}
				</div>
				<div class="relative">
					<ArrowUpDown
						size={16}
						class="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500"
					/>
					<select
						value={videoOrder.value}
						onInput={(e) => {
							videoOrder.value = (e.target as HTMLSelectElement)
								.value as OrderType;
						}}
						aria-label={t("music.playlist.orderBy")}
						class="h-10 w-max appearance-none rounded-md border border-zinc-800 bg-zinc-950 pl-9 pr-3 text-sm text-white outline-none focus:border-zinc-600"
					>
						<option value="added">{t("music.playlist.order.added")}</option>
						<option value="reverse">{t("music.playlist.order.reverse")}</option>
						<option value="duration-asc">
							{t("music.playlist.order.durationAsc")}
						</option>
						<option value="duration-desc">
							{t("music.playlist.order.durationDesc")}
						</option>
					</select>
				</div>
			</div>

			{videosQuery.isLoading ? (
				<Rows />
			) : filteredVideos.length === 0 ? (
				<p class="text-balance flex flex-1 items-center justify-center px-3 text-sm text-zinc-500 md:px-0">
					{t("pages.user.noVideos")}
				</p>
			) : (
				<Virtualization
					songs={filteredVideos}
					resetKey={`${videoOrder.value}|${query}`}
				/>
			)}
		</div>
	);

	return (
		<div class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
			<UserHeader
				source={source}
				identifier={id}
				name={infoQuery.data?.name}
				imageUrl={infoQuery.data?.avatar}
				pending={youtube && infoQuery.isLoading}
			/>

			{/* The tabs pick what the page shows, so they stand on their own above
			    the body instead of riding in a panel: the videos are a list, not a
			    shelf, and the playlists bring their own. They sit tight against the
			    body, which keeps the loose gap from the header for itself. */}
			<div class="flex min-h-0 flex-1 flex-col gap-2">
				{youtube && (
					<div class="flex shrink-0 items-center gap-2 px-3 md:px-0">
						<Tabs
							items={[
								{
									id: "videos",
									icon: CirclePlay,
									label: t("pages.youtube.videos"),
								},
								{
									id: "playlists",
									icon: ListVideo,
									label: t("pages.youtube.playlists"),
								},
							]}
							active={activeTab.value}
							onSelect={(next) => {
								activeTab.value = next;
							}}
						/>
					</div>
				)}

				{watchingVideos ? (
					videoBody
				) : (
					<Shelf
						items={items}
						to={(id) => `/playlist/${id}`}
						leading={(id) => (followable(id) ? <SaveButton id={id} /> : null)}
						actions={(id) => <PlaylistPlayButton id={id} />}
						onPlay={(id) => void playPlaylist(queryClient, id)}
						viewKey="user_view"
						isLoading={playlistsQuery.isLoading}
						emptyMessage={t("pages.user.noPlaylists")}
					/>
				)}
			</div>
		</div>
	);
}
