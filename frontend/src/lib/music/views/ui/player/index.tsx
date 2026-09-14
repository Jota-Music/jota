import { signal } from "@preact/signals";
import {
	ChevronUp,
	ListMusic,
	Repeat,
	Repeat1,
	Shuffle,
	SkipBack,
	SkipForward,
} from "lucide-preact";
import { useRef } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { usePlayer } from "@/lib/music/views/hooks/use-player";
import {
	cycleRepeat,
	playerSkeletonOn,
	repeat,
	showQueue,
	shuffle,
	toggleShuffle,
} from "@/lib/music/views/stores/queue";
import SaveYoutubeId from "@/lib/music/views/ui/player/save-youtube-id";
import { PlayerSkeleton } from "@/lib/music/views/ui/player/skeleton";
import Toggle from "@/lib/music/views/ui/player/toggle";
import VolumeControl from "@/lib/music/views/ui/volume";
import { secondsToTime } from "@/lib/shared/utils/format";
import { cn } from "@/lib/shared/utils/tw";
import AlbumLink from "@/lib/shared/views/ui/components/album-link";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import CircularProgress from "@/lib/shared/views/ui/components/circular-progress";
import Progress from "@/lib/shared/views/ui/components/progress";
import { Sheet } from "@/lib/shared/views/ui/components/sheet";

const playerModalOpen = signal(false);

interface FullPlayerProps {
	song: Song;
	cover: string;
	isLoading: boolean;
	isPlaying: boolean;
	progress: number;
	duration: number;
	seek: (seconds: number) => void;
	seekCommit: (seconds: number) => void;
	toggleSong: () => Promise<void>;
	nextSong: () => Promise<void>;
	prevSong: () => Promise<void>;
	canPrev: boolean;
	canNext: boolean;
}

function FullPlayerContent({
	song,
	cover,
	isLoading,
	isPlaying,
	progress,
	duration,
	seek,
	seekCommit,
	toggleSong,
	nextSong,
	prevSong,
	canPrev,
	canNext,
}: FullPlayerProps) {
	return (
		<section class="w-full max-w-2xl mx-auto relative z-10 flex flex-col md:flex-row gap-4 md:gap-6 items-start justify-center text-(--dominant-color)">
			<div class="relative flex max-h-max w-full md:w-max">
				<CircularProgress
					value={progress}
					max={duration}
					min={0}
					onChange={seek}
					onCommit={seekCommit}
					class="text-current mx-auto w-48 drop-shadow-lg drop-shadow-black"
				>
					<div class="relative aspect-square w-full">
						<img
							src={cover}
							alt={song.name}
							draggable={false}
							class="absolute inset-0 h-full w-full object-cover rounded-2xl shadow-lg opacity-40 animate-[spin_120s_linear_infinite] select-none"
							style="-webkit-user-drag: none"
						/>

						<div class="absolute inset-0 w-max h-max flex items-center justify-center gap-6 m-auto">
							<button
								type="button"
								disabled={!canPrev}
								onClick={() => void prevSong()}
								class="p-1 rounded-full transition drop-shadow-lg drop-shadow-black disabled:brightness-60 disabled:cursor-not-allowed cursor-pointer"
							>
								<SkipBack class="size-6 fill-current shadow-lg" />
							</button>

							<Toggle
								loading={isLoading}
								playing={isPlaying}
								onClick={() => void toggleSong()}
								class="bg-(--dominant-color)"
							/>

							<button
								type="button"
								disabled={!canNext}
								onClick={() => void nextSong()}
								class="p-1 rounded-full transition drop-shadow-lg drop-shadow-black disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
							>
								<SkipForward class="size-6 fill-current shadow-lg" />
							</button>
						</div>
					</div>
				</CircularProgress>
			</div>

			<div class="flex-1 min-w-0 w-full">
				<small class="text-white text-xs opacity-60 block truncate">
					<AlbumLink album={song.album} />
				</small>

				<h1 class="text-white text-xl font-semibold truncate">{song.name}</h1>

				<p class="text-white text-sm opacity-70 truncate">
					<ArtistLinks artists={song.artists} />
				</p>

				<SaveYoutubeId song={song} />

				<div class="mt-3 flex gap-2 justify-between items-center">
					<span class="text-white text-sm opacity-70 tabular-nums shrink-0">
						{secondsToTime(progress)}
					</span>

					<Progress
						value={progress}
						max={duration}
						min={0}
						onChange={seek}
						onCommit={seekCommit}
						class="h-1.5 min-w-0 flex-1 opacity-80"
					/>

					<span class="text-white text-sm opacity-70 tabular-nums shrink-0">
						{secondsToTime(duration)}
					</span>
				</div>

				<div class="flex items-center justify-between gap-4 mt-4 pb-2">
					<div class="flex items-center gap-2">
						<button
							type="button"
							title={
								shuffle.value ? "Desactivar aleatorio" : "Activar aleatorio"
							}
							onClick={toggleShuffle}
							aria-pressed={shuffle.value}
							class={cn(
								"rounded-md p-1 transition cursor-pointer",
								shuffle.value
									? "bg-(--dominant-color)/20 text-(--dominant-color)"
									: "text-white/60 hover:bg-white/10 hover:text-white",
							)}
						>
							<Shuffle class="size-5 stroke-current" />
						</button>

						<button
							type="button"
							title={
								repeat.value === "off"
									? "Repetir todo"
									: repeat.value === "all"
										? "Repetir uno"
										: "Sin repetición"
							}
							onClick={cycleRepeat}
							aria-label="Modo de repetición"
							aria-pressed={repeat.value !== "off"}
							class={cn(
								"rounded-md p-1 transition cursor-pointer",
								repeat.value !== "off"
									? "bg-(--dominant-color)/20 text-(--dominant-color)"
									: "text-white/60 hover:bg-white/10 hover:text-white",
							)}
						>
							{repeat.value === "one" ? (
								<Repeat1 class="size-5 stroke-current" />
							) : (
								<Repeat class="size-5 stroke-current" />
							)}
						</button>
					</div>

					<Toggle
						loading={isLoading}
						playing={isPlaying}
						onClick={() => void toggleSong()}
						iconClass="size-5"
						class="bg-(--dominant-color)"
					/>

					<VolumeControl />

					<button
						type="button"
						title="Cola de reproducción"
						aria-expanded={showQueue.value}
						onClick={() => {
							showQueue.value = !showQueue.value;
						}}
						class="rounded-md p-1 transition hover:bg-white/10 cursor-pointer"
					>
						<ListMusic class="size-6 fill-current" />
					</button>
				</div>
			</div>
		</section>
	);
}

export function Player() {
	const player = usePlayer();

	if (!player) {
		return playerSkeletonOn.value ? <PlayerSkeleton /> : null;
	}

	const {
		song,
		cover,
		isLoading,
		isPlaying,
		toggleSong,
		progress,
		duration,
		seek,
		seekCommit,
		prevSong,
		nextSong,
		canPrev,
		canNext,
	} = player;

	const dragTracking = useRef({ startY: 0, active: false });

	const onBarPointerDown = (e: PointerEvent) => {
		const target = e.target as HTMLElement;
		if (target.closest("button, input, a, [role='slider']")) return;
		dragTracking.current.active = true;
		dragTracking.current.startY = e.clientY;
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	};

	const onBarPointerMove = (e: PointerEvent) => {
		if (!dragTracking.current.active) return;
		const delta = dragTracking.current.startY - e.clientY;
		if (delta > 30) {
			dragTracking.current.active = false;
			playerModalOpen.value = true;
		}
	};

	const onBarPointerUp = () => {
		dragTracking.current.active = false;
	};

	return (
		<>
			<div class="hidden md:block">
				<FullPlayerContent {...player} />
			</div>

			<div
				class="block md:hidden fixed bottom-0 left-0 right-0 z-40 bg-stone-950 border-t border-white/10 transition-transform duration-300 ease-out"
				style={`transform: translateY(${playerModalOpen.value ? "100%" : "0"})`}
				onPointerDown={onBarPointerDown}
				onPointerMove={onBarPointerMove}
				onPointerUp={onBarPointerUp}
			>
				<div class="px-0 pt-1 text-(--dominant-color)">
					<Progress
						value={progress}
						max={duration}
						min={0}
						onChange={seek}
						onCommit={seekCommit}
						class="h-0.75 w-full rounded-none border-0 bg-neutral-800"
					/>
				</div>

				<div class="flex items-center gap-2 w-full px-4 py-3">
					<div
						role="toolbar"
						tabIndex={-1}
						onClick={() => {
							playerModalOpen.value = true;
						}}
						onKeyDown={(e) => {
							if (e.key === "Enter" || e.key === " ") {
								playerModalOpen.value = true;
							}
						}}
						class="flex items-center gap-3 flex-1 min-w-0 cursor-pointer"
					>
						<div class="relative w-12 h-12 shrink-0">
							<img
								src={cover}
								alt={song.name}
								draggable={false}
								class="w-full h-full object-cover rounded-lg shadow-lg select-none"
								style="-webkit-user-drag: none"
							/>
							<div class="absolute inset-0 rounded-lg ring-1 ring-inset ring-white/10" />
						</div>

						<div class="flex-1 min-w-0 text-left">
							<p class="text-white text-sm font-medium truncate">{song.name}</p>
							<p class="text-white/60 text-xs truncate">
								<ArtistLinks artists={song.artists} />
							</p>
						</div>
					</div>

					<div class="flex items-center gap-1 shrink-0">
						<VolumeControl class="mr-2 max-[500px]:hidden" />

						<button
							type="button"
							disabled={!canPrev}
							onClick={() => void prevSong()}
							class="p-1.5 rounded-full transition disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer text-white/80"
						>
							<SkipBack class="size-6 fill-current" />
						</button>

						<Toggle
							loading={isLoading}
							playing={isPlaying}
							onClick={() => void toggleSong()}
							class="bg-(--dominant-color)"
						/>

						<button
							type="button"
							disabled={!canNext}
							onClick={() => void nextSong()}
							class="p-1.5 rounded-full transition disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer text-white/80"
						>
							<SkipForward class="size-6 fill-current" />
						</button>

						<button
							type="button"
							aria-label="Expand player"
							onClick={(e) => {
								e.stopPropagation();
								playerModalOpen.value = true;
							}}
							class="shrink-0 p-1.5 rounded-full text-white/60 hover:text-white cursor-pointer transition-colors"
						>
							<ChevronUp class="size-5" />
						</button>
					</div>
				</div>
			</div>

			<Sheet
				open={playerModalOpen.value}
				close={() => {
					playerModalOpen.value = false;
				}}
				mobileOnly
				closeLabel="Cerrar reproductor"
			>
				<div class="min-h-0 flex-1 overflow-y-auto px-4 pt-2 pb-8">
					<FullPlayerContent {...player} />
				</div>
			</Sheet>
		</>
	);
}
