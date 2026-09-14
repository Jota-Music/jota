import { useQuery } from "@tanstack/preact-query";
import { useLayoutEffect } from "preact/hooks";
import {
	spotifyConnected,
	spotifyUser,
	syncSpotifyStatus,
} from "@/lib/auth/views/stores/session";
import getUserPlaylists from "@/lib/music/app/get-user-playlists";
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

	const spotifyHandle = spotifyUser.value ?? "default";
	const { data: playlists, isLoading } = useQuery({
		queryKey: ["user-playlists", spotifyHandle],
		queryFn: () => getUserPlaylists(spotifyHandle),
		enabled: spotifyConnected.value,
	});

	return (
		<DefaultLayout class="gap-6">
			<div class="flex flex-col gap-6 min-h-0 flex-1">
				{spotifyConnected.value && (
					<h3 class="text-base font-semibold text-zinc-200 shrink-0">
						{spotifyHandle === "default"
							? "Playlists"
							: `@${spotifyHandle}'s playlists`}
					</h3>
				)}

				{spotifyConnected.value && (
					<Shelf
						items={(playlists ?? []).map(
							(p): Item => ({
								id: p.id,
								name: p.name,
								cover: p.cover ?? p.mosaic,
							}),
						)}
						to={(id) => `/playlist/${id}`}
						isLoading={isLoading}
						emptyMessage="No playlists found."
					/>
				)}
			</div>
		</DefaultLayout>
	);
}

export default MainPage;
