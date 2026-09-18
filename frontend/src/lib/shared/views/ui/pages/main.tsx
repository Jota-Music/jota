import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { Library, ListMusic, Turntable, Users } from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import { useLocation } from "wouter-preact";
import { spotifyConnected, spotifyUser } from "@/lib/auth/views/stores/session";
import getUserPlaylists from "@/lib/music/app/get-user-playlists";
import { isLiked, likedCover } from "@/lib/music/app/liked";
import { getOrder, saveOrder } from "@/lib/music/app/order";
import { deletePlaylist, getPlaylists } from "@/lib/music/app/playlists";
import { applyOrder, move } from "@/lib/music/app/reorder";
import {
	getYouTubePlaylists,
	removeYouTubePlaylist,
} from "@/lib/music/app/youtube-playlist";
import { CreatePlaylistModal } from "@/lib/music/views/ui/playlists/create";
import {
	type IconType,
	type Item,
	Shelf,
	YouTubeHint,
} from "@/lib/music/views/ui/shelf";
import { FollowingShelf } from "@/lib/music/views/ui/user/following";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { addError } from "@/lib/shared/views/stores/errors";
import { ConfirmModal } from "@/lib/shared/views/ui/components/confirm";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";
import { RoomsShelf } from "@/lib/sync/views/ui/rooms";

type Tab = "playlists" | "following" | "rooms";

const tabKey = "main_tab";

function loadTab(): Tab {
	const saved = localStorage.getItem(tabKey);
	return saved === "following" || saved === "rooms" ? saved : "playlists";
}

export function MainPage() {
	const queryClient = useQueryClient();
	const [, setLocation] = useLocation();
	const spotifyHandle = spotifyUser.value ?? "default";
	const [tab, setTab] = useState<Tab>(loadTab);
	const [creating, setCreating] = useState(false);
	const [confirming, setConfirming] = useState<string | null>(null);

	useEffect(() => {
		localStorage.setItem(tabKey, tab);
	}, [tab]);

	const activeTab: Tab =
		tab === "following" && !spotifyConnected.value ? "playlists" : tab;

	const spotifyQuery = useQuery({
		queryKey: ["user-playlists", spotifyHandle],
		queryFn: () => getUserPlaylists(spotifyHandle),
		enabled: spotifyConnected.value,
	});

	const youtubeQuery = useQuery({
		queryKey: ["youtube-playlists"],
		queryFn: getYouTubePlaylists,
	});

	const customQuery = useQuery({
		queryKey: ["playlists"],
		queryFn: getPlaylists,
	});

	const orderQuery = useQuery({
		queryKey: ["playlist-order", spotifyHandle],
		queryFn: () => getOrder(spotifyHandle),
	});

	const removeYouTube = useMutation({
		mutationFn: removeYouTubePlaylist,
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["youtube-playlists"] }),
	});

	const removeCustom = useMutation({
		mutationFn: deletePlaylist,
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ["playlists"] }),
		onError: (error) => addError(error, "playlist"),
	});

	const items: Item[] = [
		...(spotifyQuery.data ?? []).map(
			(p): Item => ({
				id: p.id,
				name: isLiked(p.id) ? t("music.likedSongs") : p.name,
				cover: isLiked(p.id) ? likedCover : p.cover,
				source: "spotify",
			}),
		),
		...(youtubeQuery.data ?? []).map(
			(p): Item => ({
				id: p.id,
				name: p.name,
				cover: p.cover,
				subtitle: p.subtitle,
				source: "youtube",
				removable: true,
			}),
		),
		...(customQuery.data ?? []).map(
			(p): Item => ({
				id: p.id,
				name: p.name,
				icon: Library,
				source: "local",
				removable: true,
			}),
		),
	];

	const ordered = applyOrder(items, orderQuery.data ?? []);

	const reorder = (fromId: string, toId: string) => {
		const next = move(
			ordered.map((item) => item.id),
			fromId,
			toId,
		);
		queryClient.setQueryData(["playlist-order", spotifyHandle], next);
		void saveOrder(spotifyHandle, next);
	};

	const remove = (id: string) => setConfirming(id);

	const removingLocal = confirming?.startsWith("local:") ?? false;

	const confirmRemove = () => {
		if (!confirming) return;
		if (removingLocal) removeCustom.mutate(confirming);
		else removeYouTube.mutate(confirming);
		setConfirming(null);
	};

	const tabs: { id: Tab; label: string; icon: IconType }[] = [
		{ id: "playlists", label: "Playlists", icon: ListMusic },
		...(spotifyConnected.value
			? [{ id: "following" as Tab, label: "Following", icon: Users }]
			: []),
		{ id: "rooms", label: "Rooms", icon: Turntable },
	];

	return (
		<DefaultLayout class="gap-6">
			<div class="flex flex-col gap-4 min-h-0 flex-1 pb-6">
				<div class="flex shrink-0 items-center gap-2 self-start">
					<div class="flex items-center gap-1 rounded-lg border border-zinc-800 bg-zinc-950 p-1">
						{tabs.map(({ id, label, icon: Icon }) => (
							<button
								key={id}
								type="button"
								title={label}
								aria-label={label}
								onClick={() => setTab(id)}
								class={cn(
									"flex h-10 md:h-8 cursor-pointer items-center rounded-md px-3 transition-colors",
									activeTab === id
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-zinc-300",
								)}
							>
								<Icon size={16} />
							</button>
						))}
					</div>
				</div>

				{activeTab === "following" ? (
					<FollowingShelf account={spotifyHandle} />
				) : activeTab === "rooms" ? (
					<RoomsShelf />
				) : (
					<Shelf
						items={ordered}
						to={(id) => `/playlist/${id}`}
						viewKey="cover_grid_view"
						isLoading={
							spotifyQuery.isLoading ||
							youtubeQuery.isLoading ||
							customQuery.isLoading
						}
						emptyMessage={<YouTubeHint />}
						onRemove={(id) => remove(id)}
						onReorder={reorder}
						onCreate={() => setCreating(true)}
					/>
				)}
			</div>

			<CreatePlaylistModal
				open={creating}
				close={() => setCreating(false)}
				onCreated={(playlist) => {
					queryClient.invalidateQueries({ queryKey: ["playlists"] });
					setLocation(`/playlist/${playlist.id}`);
				}}
			/>

			<ConfirmModal
				open={!!confirming}
				danger={removingLocal}
				pending={removeCustom.isPending || removeYouTube.isPending}
				close={() => setConfirming(null)}
				onConfirm={confirmRemove}
			/>
		</DefaultLayout>
	);
}

export default MainPage;
