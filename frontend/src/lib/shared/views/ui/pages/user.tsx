import { useSignal } from "@preact/signals";
import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { ListMusic, ListVideo, Loader, RefreshCw } from "lucide-preact";
import { useParams } from "wouter-preact";
import { getChannelInfo, getChannelVideos } from "@/lib/music/app/get-channel";
import {
	getUserPlaylists,
	revalidateUserPlaylists,
	type UserSource,
} from "@/lib/music/app/get-user-playlists";
import { isLiked, likedCover } from "@/lib/music/app/liked";
import type { PlaylistSummary } from "@/lib/music/model";
import { playPlaylist } from "@/lib/music/views/play";
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { type Item, Shelf } from "@/lib/music/views/ui/shelf";
import { UserHeader } from "@/lib/music/views/ui/user/header";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

type Tab = "playlists" | "videos";

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

	const content =
		youtube && activeTab.value === "videos" ? (
			videosQuery.isLoading ? (
				<div class="flex items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
					<Loader size={24} class="animate-spin" />
				</div>
			) : videos.length === 0 ? (
				<p class="text-sm text-zinc-500">{t("pages.user.noVideos")}</p>
			) : (
				<Virtualization songs={videos} />
			)
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
			<div class="flex flex-col gap-6 min-h-0 flex-1">
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
