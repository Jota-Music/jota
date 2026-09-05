import "@/lib/music/views/stores/audio";

import { useQuery } from "@tanstack/preact-query";
import { Disc3, Loader } from "lucide-preact";
import { useParams } from "wouter-preact";
import { getAlbumTracks } from "@/lib/music/app/get-album";
import type { Song } from "@/lib/music/model";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function AlbumPage() {
	const { id } = useParams<{ id: string }>();
	useMeta(`Jota | Album`, `Browse and listen to an album on Jota`);

	const { data, isLoading, isError } = useQuery({
		queryKey: ["album-tracks", id],
		queryFn: () => getAlbumTracks(id ?? ""),
		enabled: !!id,
	});

	const tracks = (data as Song[]) ?? [];
	const cover = tracks[0]?.album?.covers?.[0];
	const albumName = tracks[0]?.album?.title ?? "Álbum";

	return (
		<DefaultLayout class="gap-4 h-full">
			<div class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
				<header class="flex items-center gap-4">
					{cover ? (
						<img
							src={cover}
							alt=""
							class="h-16 w-16 shrink-0 rounded-md object-cover"
						/>
					) : (
						<div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-md bg-zinc-900 text-zinc-500">
							<Disc3 size={28} />
						</div>
					)}
					<div class="min-w-0">
						<h2 class="truncate text-xl font-semibold leading-tight">
							{albumName}
						</h2>
						<p class="text-sm opacity-70 mt-1">
							{tracks.length} canción{tracks.length === 1 ? "" : "es"}
						</p>
					</div>
				</header>

				{isError ? (
					<div class="flex items-center gap-2 p-4 text-sm text-red-400">
						Failed to load album
					</div>
				) : isLoading ? (
					<div class="flex min-h-0 flex-1 items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
						<Loader size={24} class="animate-spin" />
					</div>
				) : tracks.length === 0 ? (
					<p class="text-sm text-zinc-500">No se encontraron canciones.</p>
				) : (
					<Virtualization songs={tracks} />
				)}
			</div>
		</DefaultLayout>
	);
}

export default AlbumPage;