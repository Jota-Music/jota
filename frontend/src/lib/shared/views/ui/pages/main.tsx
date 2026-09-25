import { useSignal } from "@preact/signals";
import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { Disc3 } from "lucide-preact";
import { useLocation } from "wouter-preact";
import { spotifyConnected, spotifyUser } from "@/lib/auth/views/stores/session";
import { getUserPlaylists } from "@/lib/music/app/get-user-playlists";
import { isLiked, likedCover } from "@/lib/music/app/liked";
import { getOrder, saveOrder } from "@/lib/music/app/order";
import { playlistSource } from "@/lib/music/app/playlist-source";
import { deletePlaylist, getPlaylists } from "@/lib/music/app/playlists";
import { applyOrder, move } from "@/lib/music/app/reorder";
import {
	getSavedPlaylists,
	removeSavedPlaylist,
} from "@/lib/music/app/saved-playlist";
import { playPlaylist } from "@/lib/music/views/play";
import { expireRemoval } from "@/lib/music/views/stores/removal";
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import { CreatePlaylistModal } from "@/lib/music/views/ui/playlists/create";
import { type Item, Shelf, YouTubeHint } from "@/lib/music/views/ui/shelf";
import { FollowingShelf } from "@/lib/music/views/ui/user/following";
import { t } from "@/lib/shared/i18n";
import { addError } from "@/lib/shared/views/stores/errors";
import { ConfirmModal } from "@/lib/shared/views/ui/components/confirm";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";
import { RoomsShelf } from "@/lib/sync/views/ui/rooms";

type Tab = "playlists" | "following" | "rooms";

export function MainPage() {
	const queryClient = useQueryClient();
	const [location, setLocation] = useLocation();
	const spotifyHandle = spotifyUser.value ?? "default";
	const creating = useSignal(false);
	const confirming = useSignal<string | null>(null);

	// Following covers the account's own follows (Spotify artists and YouTube
	// channels) plus local friends, so it works without a Spotify account.
	const tab: Tab = location.startsWith("/following")
		? "following"
		: location.startsWith("/rooms")
			? "rooms"
			: "playlists";

	const spotifyQuery = useQuery({
		queryKey: ["user-playlists", "spotify", spotifyHandle],
		queryFn: () => getUserPlaylists("spotify", spotifyHandle),
		enabled: spotifyConnected.value,
	});

	const savedQuery = useQuery({
		queryKey: ["saved-playlists"],
		queryFn: getSavedPlaylists,
	});

	const customQuery = useQuery({
		queryKey: ["playlists"],
		queryFn: getPlaylists,
	});

	const orderQuery = useQuery({
		queryKey: ["playlist-order", spotifyHandle],
		queryFn: () => getOrder(spotifyHandle),
	});

	const removeCustom = useMutation({
		mutationFn: deletePlaylist,
		onSuccess: (_data, id) => {
			queryClient.invalidateQueries({ queryKey: ["playlists"] });
			expireRemoval(id);
		},
		onError: (error) => addError(error, "playlist"),
	});

	const removeSaved = useMutation({
		mutationFn: removeSavedPlaylist,
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["saved-playlists"] }),
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
		...(savedQuery.data ?? []).map((p): Item => {
			const source = playlistSource(p.id);
			return {
				id: p.id,
				name: p.name,
				cover: p.cover,
				subtitle: p.subtitle,
				source,
				// YouTube playlists aren't "liked": remove them with the trash.
				removable: source === "youtube",
			};
		}),
		...(customQuery.data ?? []).map(
			(p): Item => ({
				id: p.id,
				name: p.name,
				covers: p.covers ?? [],
				icon: Disc3,
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

	const remove = (id: string) => {
		confirming.value = id;
	};

	const removingLocal = confirming.value?.startsWith("local:") ?? false;

	const confirmRemove = () => {
		if (!confirming.value) return;
		if (removingLocal) removeCustom.mutate(confirming.value);
		else removeSaved.mutate(confirming.value);
		confirming.value = null;
	};

	return (
		<DefaultLayout class="gap-6">
			<div class="flex flex-col gap-4 min-h-0 flex-1 pb-6">
				{tab === "following" ? (
					<FollowingShelf account={spotifyHandle} />
				) : tab === "rooms" ? (
					<RoomsShelf />
				) : (
					<Shelf
						items={ordered}
						to={(id) => `/playlist/${id}`}
						viewKey="cover_grid_view"
						actions={(id) => <PlaylistPlayButton id={id} />}
						isLoading={
							spotifyQuery.isLoading ||
							savedQuery.isLoading ||
							customQuery.isLoading
						}
						emptyMessage={<YouTubeHint />}
						onPlay={(id) => void playPlaylist(queryClient, id)}
						onRemove={(id) => remove(id)}
						onReorder={reorder}
						onCreate={() => {
							creating.value = true;
						}}
					/>
				)}
			</div>

			<CreatePlaylistModal
				open={creating.value}
				close={() => {
					creating.value = false;
				}}
				onCreated={(playlist) => {
					queryClient.invalidateQueries({ queryKey: ["playlists"] });
					setLocation(`/playlist/${playlist.id}`);
				}}
			/>

			<ConfirmModal
				open={!!confirming.value}
				danger={removingLocal}
				pending={removeCustom.isPending || removeSaved.isPending}
				close={() => {
					confirming.value = null;
				}}
				onConfirm={confirmRemove}
			/>
		</DefaultLayout>
	);
}
