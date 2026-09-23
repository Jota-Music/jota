import { useSignal } from "@preact/signals";
import { useQuery, useQueryClient } from "@tanstack/preact-query";
import {
	ArrowUpDown,
	ListMusic,
	ListVideo,
	Loader,
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
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { type Item, Shelf } from "@/lib/music/views/ui/shelf";
import { UserHeader } from "@/lib/music/views/ui/user/header";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

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
	const activeTab = useSignal<Tab>("playlists");
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

	if (playlistsQuery.isError || (youtube && videosQuery.isError)) {
		return (
			<DefaultLayout>
				<div class="flex flex-col items-start gap-3 p-6 text-sm">
					<p class="text-red-400">{t("pages.user.failed", { user: id })}</p>
					<button
						type="button"
						onClick={handleRefresh}
						title={t("common.retry")}
						class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white transition-colors"
					>
						<RefreshCw size={14} />
					</button>
				</div>
			</DefaultLayout>
		);
	}

	const playlists = playlistsQuery.data ?? [];
	const videos = videosQuery.data ?? [];

	const query = videosSearch.value.trim().toLowerCase();
	const base =
		query.length === 0
			? videos
			: videos.filter((video) => video.name.toLowerCase().includes(query));
	const filteredVideos = sortVideos(base, videoOrder.value);

	const content =
		youtube && activeTab.value === "videos" ? (
			<>
				<div class="grid grid-cols-[1fr_auto] gap-2">
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
					<div class="relative w-full">
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
							<option value="reverse">
								{t("music.playlist.order.reverse")}
							</option>
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
					<div class="flex items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
						<Loader size={24} class="animate-spin" />
					</div>
				) : filteredVideos.length === 0 ? (
					<p class="text-sm text-zinc-500">{t("pages.user.noVideos")}</p>
				) : (
					<Virtualization songs={filteredVideos} />
				)}
			</>
		) : (
			<Shelf
				items={playlists.map(toItem)}
				to={(id) => `/playlist/${id}`}
				actions={(id) => <PlaylistPlayButton id={id} />}
				onPlay={playPlaylist}
				viewKey="user_view"
				isLoading={playlistsQuery.isLoading}
				emptyMessage={t("pages.user.noPlaylists")}
			/>
		);

	return (
		<DefaultLayout class="gap-6">
			<div class="flex flex-col gap-6 min-h-0 flex-1 pb-6">
				<UserHeader
					source={source}
					identifier={id}
					name={infoQuery.data?.name}
					imageUrl={infoQuery.data?.avatar}
				/>

				{youtube && (
					<div class="flex items-center gap-1 self-start rounded-lg border border-zinc-800 bg-zinc-950 p-1">
						<button
							type="button"
							title={t("pages.user.videosTab")}
							onClick={() => {
								activeTab.value = "videos";
							}}
							class={cn(
								"flex h-8 cursor-pointer items-center justify-center rounded-md transition-colors px-3",
								activeTab.value === "videos"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-zinc-300",
							)}
						>
							<ListVideo size={16} />
						</button>
						<button
							type="button"
							title={t("pages.user.playlistsTab")}
							onClick={() => {
								activeTab.value = "playlists";
							}}
							class={cn(
								"flex h-8 cursor-pointer items-center justify-center rounded-md transition-colors px-3",
								activeTab.value === "playlists"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-zinc-300",
							)}
						>
							<ListMusic size={16} />
						</button>
					</div>
				)}

				{content}
			</div>
		</DefaultLayout>
	);
}
