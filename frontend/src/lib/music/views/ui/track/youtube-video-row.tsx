import type { Song } from "@/lib/music/model";
import { rowSelect } from "@/lib/music/views/hooks/row-select";
import {
	selectedSongs,
	toggleSelection,
} from "@/lib/music/views/stores/selection";
import TrackActions, {
	buildActions,
} from "@/lib/music/views/ui/track/track-actions";
import { cn } from "@/lib/shared/utils/tw";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import { openContextMenu } from "@/lib/shared/views/ui/components/context-menu";
import PlaylistCover from "@/lib/shared/views/ui/components/playlist-cover";

type Props = {
	song: Song;
	songs: Song[];
	index: number;
	onClick: () => void;
	showChannel?: boolean;
};

export function YouTubeVideoRow({
	song,
	songs,
	index,
	onClick,
	showChannel = true,
}: Props) {
	const selected = selectedSongs.value.some((s) => s.id === song.id);

	const { onClick: onRowClick } = rowSelect({
		song,
		songs,
		index,
		onActivate: onClick,
	});

	return (
		<div
			data-row={song.id}
			role="none"
			class={cn(
				"flex w-full select-none items-center gap-3 border-b border-zinc-900 px-3 py-2 transition hover:bg-zinc-800/60",
				selected && "bg-zinc-800/60",
			)}
			onContextMenu={(e) => {
				const { actions } = buildActions({
					songs: [song],
					onToggleSelect: toggleSelection,
				});
				openContextMenu(actions, e);
			}}
		>
			<button
				type="button"
				onClick={onRowClick}
				class="flex min-w-0 flex-1 cursor-pointer items-center gap-3 text-left"
			>
				<PlaylistCover
					src={song.album?.covers?.[0]}
					alt={song.name}
					imgClass="size-10 shrink-0 rounded-md object-cover"
				/>

				<div class="min-w-0 flex-1">
					<p class="wrap-break-word text-balance text-sm text-white leading-snug">
						{song.name}
					</p>
					{showChannel && (
						<p class="truncate text-xs text-zinc-500">
							<ArtistLinks artists={song.artists} />
						</p>
					)}
				</div>
			</button>

			<TrackActions
				songs={[song]}
				selection={selected}
				onToggleSelect={toggleSelection}
			/>
		</div>
	);
}
