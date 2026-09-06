import { useQuery } from "@tanstack/preact-query";
import { ListPlus, Loader } from "lucide-preact";
import { useRoute } from "wouter-preact";
import { searchYouTube, youtubeVideoToSong } from "@/lib/music/app/search";
import type { Song } from "@/lib/music/model";
import {
	enqueue,
	playFromQueueSelection,
} from "@/lib/music/views/stores/player";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";
import PlaylistCover from "@/lib/shared/views/ui/components/playlist-cover";

export function YouTubeSearchPage() {
	const [, params] = useRoute<{ query: string }>(
		"/search/youtube/:query",
	);

	const query = params?.query
		? decodeURIComponent(params.query)
		: "";

	useMeta(
		`Jota | YouTube: ${query}`,
		`YouTube results for "${query}"`,
	);

	const { data, isLoading, isError } = useQuery({
		queryKey: ["youtube-search", query],
		queryFn: () => searchYouTube(query),
		enabled: !!query,
	});

	if (isError) {
		return (
			<DefaultLayout class="gap-4">
				<div class="flex flex-col gap-4 min-h-0 flex-1">
					<header class="shrink-0">
						<h2 class="text-xl font-semibold leading-tight">
							{query}
						</h2>
					</header>
					<p class="text-red-400">Failed to search YouTube</p>
				</div>
			</DefaultLayout>
		);
	}

	const videos =
		(data as Awaited<ReturnType<typeof searchYouTube>>) ?? [];

	const songs: Song[] = videos.map(youtubeVideoToSong);

	function handleClick(song: Song) {
		void playFromQueueSelection(songs, song);
	}

	return (
		<DefaultLayout class="gap-4">
			<div class="flex flex-col gap-4 min-h-0 flex-1">
				<header class="shrink-0">
					<h2 class="text-xl font-semibold leading-tight">
						{query}
					</h2>
				</header>

				{isLoading ? (
					<div class="flex items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
						<Loader size={24} class="animate-spin" />
					</div>
				) : songs.length === 0 ? (
					<p class="text-sm text-zinc-500">No results found.</p>
				) : (
					<section class="flex flex-col gap-3 min-h-0 flex-1 overflow-y-auto pb-8">
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

function YouTubeVideoItem({
	song,
	onClick,
}: {
	song: Song;
	onClick: () => void;
}) {
	return (
		<div class="flex w-full min-w-0 items-center gap-3 rounded-md border border-zinc-800 bg-zinc-900/50 p-3 transition-colors hover:bg-zinc-800/50">
			<button
				type="button"
				onClick={onClick}
				class="flex min-w-0 flex-1 cursor-pointer items-center gap-3 text-left"
			>
				<PlaylistCover
					src={song.album?.covers?.[0]}
					alt={song.name}
					imgClass="size-10 shrink-0 rounded-md object-cover"
				/>

				<div class="min-w-0 flex-1">
					<p class="wrap-break-word text-sm font-medium text-zinc-200">
						{song.name}
					</p>

					<p class="truncate text-xs text-zinc-500">
						YouTube
					</p>
				</div>
			</button>

			<button
				type="button"
				title="Añadir a la cola"
				onClick={() => enqueue(song)}
				class="shrink-0 cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800/80 hover:text-amber-300"
			>
				<ListPlus size={18} strokeWidth={2} />
			</button>
		</div>
	);
}

export default YouTubeSearchPage;
