import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { spotifyConnected, spotifyUser } from "@/lib/auth/views/stores/session";
import getUserPlaylists from "@/lib/music/app/get-user-playlists";
import {
	getYouTubePlaylists,
	removeYouTubePlaylist,
} from "@/lib/music/app/youtube-playlist";
import { type Item, Shelf, YouTubeHint } from "@/lib/music/views/ui/shelf";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function MainPage() {
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
					Playlists
				</h3>

				<Shelf
					items={items}
					to={(id) => `/playlist/${id}`}
					isLoading={spotifyQuery.isLoading || youtubeQuery.isLoading}
					emptyMessage={<YouTubeHint />}
					onRemove={(id) => remove.mutate(id)}
				/>
			</div>
		</DefaultLayout>
	);
}

export default MainPage;
