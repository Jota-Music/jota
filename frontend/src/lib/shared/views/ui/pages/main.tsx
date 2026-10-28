import { GetMyYouTubePlaylists } from "@bindings/app";
import { useSignal } from "@preact/signals";
import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { Disc3 } from "lucide-preact";
import { useMemo } from "preact/hooks";
import { useLocation } from "wouter-preact";
import {
	spotifyConnected,
	spotifyDegraded,
	spotifyReady,
	spotifyUser,
} from "@/lib/auth/views/stores/session";
import { youtubeSignedIn } from "@/lib/auth/views/stores/youtube";
import { getUserPlaylists } from "@/lib/music/app/get-user-playlists";
import { isLiked, likedCover } from "@/lib/music/app/liked";
import { playlistSource } from "@/lib/music/app/playlist-source";
import { deletePlaylist, getPlaylists } from "@/lib/music/app/playlists";
import { playPlaylist } from "@/lib/music/views/play";
import { expireRemoval } from "@/lib/music/views/stores/removal";
import { SaveButton } from "@/lib/music/views/ui/follow";
import { PlaylistPlayButton } from "@/lib/music/views/ui/playlist/play-button";
import { followable, useSaved } from "@/lib/music/views/ui/playlist/saved";
import { CreatePlaylistModal } from "@/lib/music/views/ui/playlists/create";
import { AllHint, type Item, Shelf } from "@/lib/music/views/ui/shelf";
import { FollowingShelf } from "@/lib/music/views/ui/user/following";
import { getOrder, saveOrder } from "@/lib/shared/app/order";
import { applyOrder, move } from "@/lib/shared/app/reorder";
import { t } from "@/lib/shared/i18n";
import { addError } from "@/lib/shared/views/stores/errors";
import { ConfirmModal } from "@/lib/shared/views/ui/components/confirm";
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
		enabled: spotifyConnected.value || spotifyDegraded.value,
	});

	// The YouTube account's own playlists sit next to Spotify's, in the same
	// shelf: the signed-in account is a source of playlists, not a setting.
	const youtubeQuery = useQuery({
		queryKey: ["user-playlists", "youtube"],
		queryFn: () => GetMyYouTubePlaylists(),
		enabled: youtubeSignedIn.value,
	});

	const { saved, isLoading: loadingSaved } = useSaved();

	const customQuery = useQuery({
		queryKey: ["playlists"],
		queryFn: getPlaylists,
	});

	const orderQuery = useQuery({
		queryKey: ["shelf-order", "library", spotifyHandle],
		queryFn: () => getOrder("library", spotifyHandle),
		enabled: spotifyReady.value,
	});

	const removeCustom = useMutation({
		mutationFn: deletePlaylist,
		onSuccess: (_data, id) => {
			queryClient.invalidateQueries({ queryKey: ["playlists"] });
			expireRemoval(id);
		},
		onError: (error) => addError(error, "playlist"),
	});

	// A playlist the account already owns is listed by the account queries
	// above, so keeping it out of the saved list avoids showing it twice.
	const followed = useMemo(() => new Set(saved.map((p) => p.id)), [saved]);

	const items = useMemo<Item[]>(() => {
		const own = new Set(
			[...(spotifyQuery.data ?? []), ...(youtubeQuery.data ?? [])].map(
				(p) => p.id,
			),
		);

		return [
			...(spotifyQuery.data ?? []).map(
				(p): Item => ({
					id: p.id,
					name: isLiked(p.id) ? t("music.likedSongs") : p.name,
					cover: isLiked(p.id) ? likedCover : p.cover,
					source: "spotify",
				}),
			),
			...(youtubeQuery.data ?? [])
				.filter((p) => !followed.has(p.id))
				.map(
					(p): Item => ({
						id: p.id,
						name: p.name,
						cover: p.cover,
						subtitle: p.subtitle,
						source: "youtube",
					}),
				),
			// The heart is what unfollows a saved playlist, so it needs no trash:
			// the shelf keeps one way to drop an external playlist and one to delete
			// a local one.
			...saved
				.filter((p) => !own.has(p.id))
				.map(
					(p): Item => ({
						id: p.id,
						name: p.name,
						cover: p.cover,
						subtitle: p.subtitle,
						source: playlistSource(p.id),
					}),
				),
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
	}, [spotifyQuery.data, youtubeQuery.data, saved, followed, customQuery.data]);

	const ordered = useMemo(
		() => applyOrder(items, orderQuery.data ?? []),
		[items, orderQuery.data],
	);

	const reorder = (fromId: string, toId: string) => {
		const next = move(
			ordered.map((item) => item.id),
			fromId,
			toId,
		);
		queryClient.setQueryData(["shelf-order", "library", spotifyHandle], next);
		void saveOrder("library", spotifyHandle, next);
	};

	const remove = (id: string) => {
		confirming.value = id;
	};

	const confirmRemove = () => {
		if (!confirming.value) return;
		removeCustom.mutate(confirming.value);
		confirming.value = null;
	};

	return (
		<>
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
						leading={(id) =>
							followed.has(id) && followable(id) ? <SaveButton id={id} /> : null
						}
						actions={(id) => <PlaylistPlayButton id={id} />}
						isLoading={
							spotifyQuery.isLoading ||
							youtubeQuery.isLoading ||
							loadingSaved ||
							customQuery.isLoading ||
							orderQuery.isLoading
						}
						emptyMessage={<AllHint />}
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
				danger
				pending={removeCustom.isPending}
				close={() => {
					confirming.value = null;
				}}
				onConfirm={confirmRemove}
			/>
		</>
	);
}
