import { ListMusic, SkipBack, SkipForward } from "lucide-preact";
import { usePlayer } from "@/lib/music/views/hooks/use-player";
import { playerSkeletonOn, showQueue } from "@/lib/music/views/stores/queue";
import SaveYoutubeId from "@/lib/music/views/ui/player/save-youtube-id";
import { PlayerSkeleton } from "@/lib/music/views/ui/player/skeleton";
import Toggle from "@/lib/music/views/ui/player/toggle";
import VolumeControl from "@/lib/music/views/ui/volume";
import { secondsToTime } from "@/lib/shared/utils/format";
import CircularProgress from "@/lib/shared/views/ui/components/circular-progress";
import Progress from "@/lib/shared/views/ui/components/progress";

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
		progress,
		duration,
		seek,
		toggleSong,
		nextSong,
		prevSong,
		canPrev,
		canNext,
	} = player;

	return (
		<section class="w-full max-w-2xl mx-auto relative z-10 flex flex-col md:flex-row gap-4 md:gap-6 items-start justify-center text-(--dominant-color)">
			<div class="relative flex max-h-max w-full md:w-max">
				<CircularProgress
					value={progress}
					max={duration}
					min={0}
					onChange={seek}
					class="text-current mx-auto w-52 drop-shadow-lg drop-shadow-black"
				>
					<div class="relative aspect-square w-full">
						<img
							src={cover}
							alt={song.name}
							class="absolute inset-0 h-full w-full object-cover rounded-2xl shadow-lg opacity-40 animate-[spin_120s_linear_infinite]"
						/>

						<div class="absolute inset-0 w-max h-max flex items-center justify-center gap-6 m-auto">
							<button
								type="button"
								disabled={!canPrev}
								onClick={() => void prevSong()}
								class="p-1 rounded-full transition drop-shadow-lg drop-shadow-black disabled:brightness-60 disabled:cursor-not-allowed cursor-pointer"
							>
								<SkipBack size={25} class="fill-current shadow-lg" />
							</button>

							<Toggle
								loading={isLoading}
								playing={isPlaying}
								onClick={() => void toggleSong()}
								size={25}
								class="bg-(--dominant-color)"
							/>

							<button
								type="button"
								disabled={!canNext}
								onClick={() => void nextSong()}
								class="p-1 rounded-full transition drop-shadow-lg drop-shadow-black disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
							>
								<SkipForward size={25} class="fill-current shadow-lg" />
							</button>
						</div>
					</div>
				</CircularProgress>
			</div>

			<div class="flex-1 min-w-0 w-full">
				<small class="text-white text-xs opacity-60 block truncate">
					{song.album.title}
				</small>

				<h1 class="text-white text-xl font-semibold truncate">{song.name}</h1>

				<p class="text-white text-sm opacity-70 truncate">
					{song.artists.map((a) => a.name).join(", ")}
				</p>

				<SaveYoutubeId song={song} isPlaying={isPlaying} />

				<div class="mt-3 flex gap-2 justify-between items-center">
					<span class="text-white text-sm opacity-70 tabular-nums shrink-0">
						{secondsToTime(progress)}
					</span>

					<Progress
						value={progress}
						max={duration}
						min={0}
						onChange={seek}
						class="h-1.5 min-w-0 flex-1 opacity-80"
					/>

					<span class="text-white text-sm opacity-70 tabular-nums shrink-0">
						{secondsToTime(duration)}
					</span>
				</div>

				<div class="flex items-center justify-between gap-4 mt-4 pb-2">
					<Toggle
						loading={isLoading}
						playing={isPlaying}
						onClick={() => void toggleSong()}
						size={18}
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
						<ListMusic size={25} class="fill-current" />
					</button>
				</div>
			</div>
		</section>
	);
}
