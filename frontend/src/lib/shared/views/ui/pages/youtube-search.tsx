import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { CirclePlay, Heart, ListPlus, ListVideo, Loader } from "lucide-preact";
import { useState } from "preact/hooks";
import { Link, useRoute } from "wouter-preact";
import {
	searchYouTube,
	searchYouTubePlaylists,
	youtubeVideoToSong,
} from "@/lib/music/app/search";
import {
	addYouTubePlaylist,
	getYouTubePlaylists,
	removeYouTubePlaylist,
} from "@/lib/music/app/youtube-playlist";
import type { PlaylistSummary, Song } from "@/lib/music/model";
import {
	enqueue,
	playFromQueueSelection,
} from "@/lib/music/views/stores/player";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import PlaylistCover from "@/lib/shared/views/ui/components/playlist-cover";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

type Tab = "videos" | "playlists";

export function YouTubeSearchPage() {
	const [, params] = useRoute<{ query: string }>("/search/youtube/:query");

	const query = params?.query ? decodeURIComponent(params.query) : "";

	useMeta(`Jota | YouTube: ${query}`, `YouTube results for "${query}"`);

	const [tab, setTab] = useState<Tab>("videos");
	const queryClient = useQueryClient();

	const videosQuery = useQuery({
		queryKey: ["youtube-search", query],
		queryFn: () => searchYouTube(query),
		enabled: !!query && tab === "videos",
	});

	const playlistsQuery = useQuery({
		queryKey: ["youtube-playlist-search", query],
		queryFn: () => searchYouTubePlaylists(query),
		enabled: !!query && tab === "playlists",
	});

	const savedQuery = useQuery({
		queryKey: ["youtube-playlists"],
		queryFn: getYouTubePlaylists,
		enabled: tab === "playlists",
	});

	const saved = new Set((savedQuery.data ?? []).map((p) => p.id));
	const [overrides, setOverrides] = useState<Record<string, boolean>>({});

	function isSaved(id: string): boolean {
		return overrides[id] ?? saved.has(id);
	}

	const toggle = useMutation({
		mutationFn: async ({ id, next }: { id: string; next: boolean }) => {
			if (next) {
				await addYouTubePlaylist(id);
			} else {
				await removeYouTubePlaylist(id);
			}
		},
		onMutate: ({ id, next }) => {
			setOverrides((o) => ({ ...o, [id]: next }));
			return { id, prev: !next };
		},
		onError: (_error, _vars, ctx) => {
			if (ctx) setOverrides((o) => ({ ...o, [ctx.id]: ctx.prev }));
		},
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["youtube-playlists"] }),
	});

	const songs: Song[] = (videosQuery.data ?? []).map(youtubeVideoToSong);
	const playlists = playlistsQuery.data ?? [];

	function handleClick(song: Song) {
		void playFromQueueSelection(songs, song);
	}

	return (
		<DefaultLayout class="gap-4">
			<div class="flex flex-col gap-4 min-h-0 flex-1 pb-6">
				<header class="flex shrink-0 flex-col gap-3">
					<h2 class="text-xl font-semibold leading-tight">{query}</h2>
					<div class="flex items-center gap-1 self-start rounded-lg border border-zinc-800 bg-zinc-950 p-1">
						<button
							type="button"
							title="Videos"
							onClick={() => setTab("videos")}
							class={`flex h-8 w-8 cursor-pointer items-center justify-center rounded-md transition-colors ${
								tab === "videos"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-zinc-300"
							}`}
						>
							<CirclePlay size={18} />
						</button>
						<button
							type="button"
							title="Playlists"
							onClick={() => setTab("playlists")}
							class={`flex h-8 w-8 cursor-pointer items-center justify-center rounded-md transition-colors ${
								tab === "playlists"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-zinc-300"
							}`}
						>
							<ListVideo size={16} />
						</button>
					</div>
				</header>

				{tab === "playlists" ? (
					<section class="flex min-h-0 flex-1 flex-col overflow-y-auto">
						{playlistsQuery.isLoading ? (
							<div class="flex items-center gap-2 text-sm text-zinc-500">
								<Loader size={14} class="animate-spin" />
								Loading...
							</div>
						) : playlistsQuery.isError ? (
							<p class="text-red-400">Failed to search YouTube</p>
						) : playlists.length === 0 ? (
							<p class="text-sm text-zinc-500">No playlists found.</p>
						) : (
							<div class="grid grid-cols-2 gap-3 pt-1 md:grid-cols-3 lg:grid-cols-4 md:gap-4">
								{playlists.map((playlist) => (
									<PlaylistCard
										key={playlist.id}
										playlist={playlist}
										saved={isSaved(playlist.id)}
										onToggle={() =>
											toggle.mutate({
												id: playlist.id,
												next: !isSaved(playlist.id),
											})
										}
									/>
								))}
							</div>
						)}
					</section>
				) : videosQuery.isLoading ? (
					<div class="flex items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
						<Loader size={24} class="animate-spin" />
					</div>
				) : videosQuery.isError ? (
					<p class="text-red-400">Failed to search YouTube</p>
				) : songs.length === 0 ? (
					<p class="text-sm text-zinc-500">No results found.</p>
				) : (
					<section class="flex min-h-0 flex-1 flex-col overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950">
						{songs.map((song) => (
							<YouTubeVideoItem
								key={song.id}
								song={song}
								onClick={() => handleClick(song)}
							/>
						))}
					</section>
				)}
			</div>
		</DefaultLayout>
	);
}

function PlaylistCard({
	playlist,
	saved,
	onToggle,
}: {
	playlist: PlaylistSummary;
	saved: boolean;
	onToggle: () => void;
}) {
	return (
		<div class="group flex flex-col gap-2">
			<div class="relative aspect-square w-full overflow-hidden rounded-md bg-zinc-900">
				<Link href={`/playlist/${playlist.id}`}>
					<PlaylistCover
						src={playlist.cover ?? playlist.mosaic}
						alt={playlist.name}
						imgClass="h-full w-full object-cover transition-opacity group-hover:opacity-80"
					/>
				</Link>

				<button
					type="button"
					title={saved ? "Remove from home" : "Add to home"}
					onClick={(e) => {
						e.preventDefault();
						e.stopPropagation();
						onToggle();
					}}
					class={`absolute right-2 top-2 flex h-8 w-8 cursor-pointer items-center justify-center rounded-full bg-black/70 backdrop-blur transition hover:bg-black/90 ${
						saved ? "text-red-400" : "text-zinc-300 hover:text-white"
					}`}
				>
					<Heart size={16} class={saved ? "fill-current" : ""} />
				</button>
			</div>

			<Link href={`/playlist/${playlist.id}`}>
				<div class="flex flex-col">
					<h3 class="truncate text-sm font-medium text-zinc-300 transition-colors group-hover:text-white">
						{playlist.name}
					</h3>
					{playlist.subtitle && (
						<p class="truncate text-xs text-zinc-500">{playlist.subtitle}</p>
					)}
				</div>
			</Link>
		</div>
	);
}

function YouTubeVideoItem({
	song,
	onClick,
}: {
	song: Song;
	onClick: () => void;
}) {
	return (
		<button
			type="button"
			onClick={onClick}
			class="flex w-full items-center gap-3 border-b border-zinc-900 px-3 py-2 transition hover:cursor-pointer hover:bg-zinc-900/40"
		>
			<PlaylistCover
				src={song.album?.covers?.[0]}
				alt={song.name}
				imgClass="size-10 shrink-0 rounded-md object-cover"
			/>

			<div class="min-w-0 flex-1 text-left">
				<p class="wrap-break-word text-balance text-sm text-white leading-snug">
					{song.name}
				</p>
			</div>

			<button
				type="button"
				title="Add to queue"
				onClick={(e) => {
					e.stopPropagation();
					enqueue(song);
				}}
				class="shrink-0 cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800/80 hover:text-amber-300"
			>
				<ListPlus size={18} strokeWidth={2} />
			</button>
		</button>
	);
}

export default YouTubeSearchPage;
