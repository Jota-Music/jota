import { computed } from "@preact/signals";
import { memo } from "preact/compat";
import type { Song } from "@/lib/music/model";
import { useWindow } from "@/lib/music/views/hooks/use-window";
import {
	currentSong,
	isLoading,
	isPlaying,
} from "@/lib/music/views/stores/audio";
import { playFromQueueSelection } from "@/lib/music/views/stores/player";
import Enqueue from "@/lib/music/views/ui/components/enqueue";
import TrackArt from "@/lib/music/views/ui/components/track-art";
import SaveYoutubeId from "@/lib/music/views/ui/player/save-youtube-id";
import { t } from "@/lib/shared/i18n";
import { secondsToTime } from "@/lib/shared/utils/format";
import { cn } from "@/lib/shared/utils/tw";
import AlbumLink from "@/lib/shared/views/ui/components/album-link";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

type Props = {
	songs: Song[];
};

const ROW_PX = 64;

const currentId = computed(() => currentSong.value?.id);

function PlaylistRow({ song, songs }: { song: Song; songs: Song[] }) {
	const isCurrent = song.id === currentId.value;

	return (
		<div
			data-id={song.id}
			role="none"
			onClick={() => void playFromQueueSelection(songs, song)}
			className={cn(
				"absolute left-0 flex h-full w-full items-center justify-between gap-2 border-b border-zinc-900 px-3 transition hover:cursor-pointer hover:bg-zinc-900/40",
				isCurrent ? "font-medium text-(--dominant-color)!" : "",
			)}
		>
			<div class="grid grid-cols-[auto_1fr] gap-2">
				<TrackArt
					song={song}
					current={isCurrent}
					loading={isLoading.value}
					playing={isPlaying.value}
					alt={song.name}
					class={
						isCurrent
							? "rounded-md outline-2 outline-(--dominant-color)"
							: undefined
					}
					imgClass="h-10 w-10 rounded-md"
				/>

				<div class="flex min-w-0 flex-col text-start">
					<span className="truncate text-sm text-white">{song.name}</span>

					<span className="truncate text-xs text-zinc-400">
						<ArtistLinks artists={song.artists} />
						{song.album?.title && (
							<>
								{" • "}
								<AlbumLink album={song.album} />
							</>
						)}
					</span>
				</div>
			</div>

			<div class="flex shrink-0 items-center gap-2">
				<SaveYoutubeId song={song} compact />
				<Enqueue song={song} title={t("music.track.enqueueAfter")} />
				<div className="tabular-nums text-xs text-zinc-500">
					{secondsToTime(song.duration)}
				</div>
			</div>
		</div>
	);
}

const PlaylistRowMemo = memo(PlaylistRow);

export function Virtualization({ songs }: Props) {
	const { ref, totalSize, items } = useWindow<HTMLDivElement>(
		songs.length,
		ROW_PX,
	);

	return (
		<div className="relative min-h-0 flex-1">
			<div
				ref={ref}
				className="h-full overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950"
			>
				{songs.length === 0 ? (
					<div className="flex min-h-32 flex-1 items-center justify-center px-4 py-8 text-sm text-zinc-500">
						{t("music.track.noSongs")}
					</div>
				) : (
					<div
						className="relative w-full pb-3"
						style={{
							height: `${totalSize}px`,
						}}
					>
						{items.map((virtualRow) => {
							const song = songs[virtualRow.index];

							return (
								<div
									key={`${song.id}-${virtualRow.index}`}
									class="absolute left-0 w-full"
									style={{
										height: `${virtualRow.size}px`,
										transform: `translateY(${virtualRow.start}px)`,
									}}
								>
									<PlaylistRowMemo song={song} songs={songs} />
								</div>
							);
						})}
					</div>
				)}
			</div>

			<Scrollbar target={ref} />
		</div>
	);
}
