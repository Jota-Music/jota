import "@/lib/music/views/stores/audio";

import { useQuery } from "@tanstack/preact-query";
import { Loader } from "lucide-preact";
import { useParams } from "wouter-preact";
import { getAlbumTracks } from "@/lib/music/app/get-album";
import type { Song } from "@/lib/music/model";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { PageHeader } from "@/lib/shared/views/ui/components/page-header";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function AlbumPage() {
	const { id } = useParams<{ id: string }>();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["album-tracks", id],
		queryFn: () => getAlbumTracks(id ?? ""),
		enabled: !!id,
	});

	const tracks = (data as Song[]) ?? [];
	const cover = tracks[0]?.album?.covers?.[0];
	const albumName = tracks[0]?.album?.title ?? "Album";

	return (
		<DefaultLayout class="gap-4">
			<div class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden pb-6">
				<PageHeader
					cover={cover}
					title={albumName}
					subtitle={`${tracks.length} track${tracks.length === 1 ? "" : "s"}`}
				/>

				{isError ? (
					<div class="flex items-center gap-2 p-4 text-sm text-red-400">
						Failed to load album
					</div>
				) : isLoading ? (
					<div class="flex min-h-0 flex-1 items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
						<Loader size={24} class="animate-spin" />
					</div>
				) : tracks.length === 0 ? (
					<p class="text-sm text-zinc-500">No tracks found.</p>
				) : (
					<Virtualization songs={tracks} />
				)}
			</div>
		</DefaultLayout>
	);
}

export default AlbumPage;
