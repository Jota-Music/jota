import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { ListMusic, Users } from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import { spotifyConnected, spotifyUser } from "@/lib/auth/views/stores/session";
import getUserPlaylists from "@/lib/music/app/get-user-playlists";
import {
	getYouTubePlaylists,
	removeYouTubePlaylist,
} from "@/lib/music/app/youtube-playlist";
import {
	type IconType,
	type Item,
	Shelf,
	YouTubeHint,
} from "@/lib/music/views/ui/shelf";
import { FollowingShelf } from "@/lib/music/views/ui/user/following";
import { cn } from "@/lib/shared/utils/tw";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

type Tab = "playlists" | "following";

const tabKey = "main_tab";

function loadTab(): Tab {
	return localStorage.getItem(tabKey) === "following"
		? "following"
		: "playlists";
}

export function MainPage() {
	const queryClient = useQueryClient();
	const spotifyHandle = spotifyUser.value ?? "default";
	const [tab, setTab] = useState<Tab>(loadTab);

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

	const tabs: { id: Tab; label: string; icon: IconType }[] = [
		{ id: "playlists", label: "Playlists", icon: ListMusic },
		...(spotifyConnected.value
			? [{ id: "following" as Tab, label: "Following", icon: Users }]
			: []),
	];

	return (
		<DefaultLayout class="gap-6">
			<div class="flex flex-col gap-4 min-h-0 flex-1 pb-6">
				<div class="flex shrink-0 items-center gap-1 self-start rounded-lg border border-zinc-800 bg-zinc-950 p-1">
					{tabs.map(({ id, label, icon: Icon }) => (
						<button
							key={id}
							type="button"
							title={label}
							aria-label={label}
							onClick={() => setTab(id)}
							class={cn(
								"flex h-8 cursor-pointer items-center rounded-md px-3 transition-colors",
								activeTab === id
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-zinc-300",
							)}
						>
							<Icon size={16} />
						</button>
					))}
				</div>

				{activeTab === "following" ? (
					<FollowingShelf account={spotifyHandle} />
				) : (
					<Shelf
						items={items}
						to={(id) => `/playlist/${id}`}
						isLoading={spotifyQuery.isLoading || youtubeQuery.isLoading}
						emptyMessage={<YouTubeHint />}
						onRemove={(id) => remove.mutate(id)}
					/>
				)}
			</div>
		</DefaultLayout>
	);
}

export default MainPage;
