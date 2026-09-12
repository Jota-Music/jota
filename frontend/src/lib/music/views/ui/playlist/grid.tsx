import { Link } from "wouter-preact";
import { PlaylistCover } from "@/lib/shared/views/ui/components/playlist-cover";
import type { PlaylistSummary } from "@/lib/music/model";

interface PlaylistGridProps {
	playlists: PlaylistSummary[];
	isLoading?: boolean;
	emptyMessage?: string;
}

export function PlaylistGrid({
	playlists,
	isLoading = false,
	emptyMessage = "No playlists found.",
}: PlaylistGridProps) {
	if (isLoading) {
		return (
			<div class="flex items-center gap-2 text-sm text-zinc-500">
				Loading playlists...
			</div>
		);
	}

	if (!playlists || playlists.length === 0) {
		return <p class="text-sm text-zinc-500">{emptyMessage}</p>;
	}

	return (
		<div class="grid grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
			{playlists.map((p) => (
				<Link key={p.id} href={`/playlist/${p.id}`}>
					<div class="rounded-md overflow-hidden cursor-pointer group">
						<div class="flex flex-col gap-2">
							<div class="aspect-square w-full overflow-hidden rounded-md">
								<PlaylistCover
									src={"mosaic" in p ? p.mosaic : p.cover}
									alt={p.name}
									imgClass="w-full h-full object-cover group-hover:opacity-80 transition-opacity"
								/>
							</div>
							<h3 class="font-medium truncate text-sm text-zinc-300 group-hover:text-white transition-colors">
								{p.name}
							</h3>
						</div>
					</div>
				</Link>
			))}
		</div>
	);
}