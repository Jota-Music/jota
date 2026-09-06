import { useVirtualizer } from "@tanstack/react-virtual";
import { ListPlus, Loader, Music, Pause } from "lucide-preact";
import { useRef } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import {
	currentSong,
	isLoading,
	isPlaying,
} from "@/lib/music/views/stores/audio";
import {
	enqueue,
	playFromQueueSelection,
} from "@/lib/music/views/stores/player";
import { secondsToTime } from "@/lib/shared/utils/format";
import { cn } from "@/lib/shared/utils/tw";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";

type Props = {
	songs: Song[];
};

const ROW_PX = 64;

export function Virtualization({ songs }: Props) {
	const parentRef = useRef<HTMLDivElement>(null);
	const current = currentSong.value;
	const loading = isLoading.value;
	const playing = isPlaying.value;

	const rowVirtualizer = useVirtualizer({
		count: songs.length,
		getScrollElement: () => parentRef.current,
		estimateSize: () => ROW_PX,
		overscan: 12,
		getItemKey: (index) => songs[index]?.id ?? index,
	});

	return (
		<div
			ref={parentRef}
			className="min-h-0 flex-1 overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950"
		>
			{songs.length === 0 ? (
				<div className="flex min-h-32 flex-1 items-center justify-center px-4 py-8 text-sm text-zinc-500">
					Sin canciones
				</div>
			) : (
			<div
				className="relative w-full pb-3"
				style={{
					height: `${rowVirtualizer.getTotalSize()}px`,
				}}
			>
					{rowVirtualizer.getVirtualItems().map((virtualRow) => {
						const song = songs[virtualRow.index];
						const isCurrent = song.id === current?.id;

						return (
							<button
								type="button"
								key={`${song.id}-${virtualRow.index}`}
								data-id={song.id}
								onClick={() => void playFromQueueSelection(songs, song)}
								className={cn(
									"absolute left-0 flex w-full items-center justify-between gap-2 border-b border-zinc-900 px-3 transition hover:cursor-pointer hover:bg-zinc-900/40",
									isCurrent ? "font-medium text-(--dominant-color)!" : "",
								)}
								style={{
									height: `${virtualRow.size}px`,
									transform: `translateY(${virtualRow.start}px)`,
								}}
							>
								<div class="grid grid-cols-[auto_1fr] gap-2">
									<div
										class={cn(
											"relative",
											isCurrent &&
												"rounded-md outline-2 outline-(--dominant-color)",
										)}
									>
										{isCurrent &&
											(loading ? (
												<Loader
													size={23}
													class="absolute inset-0 z-10 m-auto animate-spin text-zinc-400 drop-shadow-md drop-shadow-black"
												/>
											) : playing ? (
												<Music
													size={25}
													class="absolute inset-0 z-10 m-auto text-(--dominant-color) drop-shadow-md drop-shadow-black"
												/>
											) : (
												<Pause
													size={25}
													class="absolute inset-0 z-10 m-auto fill-(--dominant-color) drop-shadow-md drop-shadow-black"
												/>
											))}

										<img
											loading="lazy"
											decoding="async"
											src={song.album?.covers?.[0]}
											alt={song.name}
											className={cn(
												"h-10 w-10 rounded-md",
												isCurrent && "brightness-40",
											)}
										/>
									</div>

									<div class="flex min-w-0 flex-col text-start">
										<span className="truncate text-sm text-white">
											{song.name}
										</span>

										<span className="truncate text-xs text-zinc-400">
											<ArtistLinks artists={song.artists} />
										</span>
									</div>
								</div>

								<div class="flex shrink-0 items-center gap-2">
									<button
										type="button"
										title="Encolar después del tema actual"
										onClick={(e) => {
											e.stopPropagation();
											enqueue(song);
										}}
										class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800/80 hover:text-amber-300"
									>
										<ListPlus size={18} strokeWidth={2} />
									</button>
									<div className="tabular-nums text-xs text-zinc-500">
										{secondsToTime(song.duration)}
									</div>
								</div>
							</button>
						);
					})}
				</div>
			)}
		</div>
	);
}
