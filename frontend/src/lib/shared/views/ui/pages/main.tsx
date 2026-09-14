import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { useLayoutEffect } from "preact/hooks";
import {
	spotifyConnected,
	spotifyUser,
	syncSpotifyStatus,
} from "@/lib/auth/views/stores/session";
import getUserPlaylists from "@/lib/music/app/get-user-playlists";
import {
	getYouTubePlaylists,
	removeYouTubePlaylist,
} from "@/lib/music/app/youtube-playlist";
import { type Item, Shelf } from "@/lib/music/views/ui/shelf";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function MainPage() {
	useMeta(
		"Jota | Free music self-hosted service",
		"Jota is a free, self-hosted music service. Stream your music anywhere, anytime.",
	);

	useLayoutEffect(() => {
		void syncSpotifyStatus();
	}, []);

	const queryClient = useQueryClient();
	const spotifyHandle = spotifyUser.value ?? "default";

	const spotifyQuery = useQuery({
		queryKey: ["user-playlists", spotifyHandle],
		queryFn: () => getUserPlaylists(spotifyHandle),
		enabled: spotifyConnected.value,
	});

	const youtubeQuery = useQuery({
		queryKey: ["youtube-playlists"],
		queryFn: getYouTubePlaylists,
	});

	const remove = useMutation({
		mutationFn: removeYouTubePlaylist,
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["youtube-playlists"] }),
	});

	const items: Item[] = [
		...(spotifyQuery.data ?? []).map(
			(p): Item => ({
				id: p.id,
				name: p.name,
				cover: p.cover ?? p.mosaic,
				source: "spotify",
			}),
		),
		...(youtubeQuery.data ?? []).map(
			(p): Item => ({
				id: p.id,
				name: p.name,
				cover: p.cover ?? p.mosaic,
				subtitle: p.subtitle,
				source: "youtube",
				removable: true,
			}),
		),
	];

	return (
		<DefaultLayout class="gap-6">
			<div class="flex flex-col gap-6 min-h-0 flex-1 pb-6">
				<h3 class="text-base font-semibold text-zinc-200 shrink-0">
					{spotifyHandle === "default"
						? "Playlists"
						: `@${spotifyHandle}'s playlists`}
				</h3>

				<Shelf
					items={items}
					to={(id) => `/playlist/${id}`}
					isLoading={spotifyQuery.isLoading || youtubeQuery.isLoading}
					emptyMessage="No playlists found. Search for YouTube playlists in the search bar."
					onRemove={(id) => remove.mutate(id)}
				/>
			</div>
		</DefaultLayout>
	);
}

export default MainPage;
