import { ListMusic, SkipBack, SkipForward } from "lucide-preact";
import Toggle from "@/lib/music/views/ui/player/toggle";
import VolumeControl from "@/lib/music/views/ui/volume";
import CircularProgress from "@/lib/shared/views/ui/components/circular-progress";
import Progress from "@/lib/shared/views/ui/components/progress";

export function PlayerSkeleton() {
	return (
		<section class="min-w-full w-full max-w-2xl mx-auto relative z-10 flex flex-col sm:flex-row gap-4 md:gap-6 items-start justify-center text-(--dominant-color) animate-pulse">
			{/* LEFT: COVER - mirror <Player /> layout to avoid layout shift */}
			<div class="relative flex max-h-max pointer-events-none">
				<CircularProgress
					value={0}
					max={1}
					min={0}
					class="text-current w-52 drop-shadow-lg drop-shadow-black"
				>
					<div class="relative aspect-square w-full">
						<div class="absolute inset-0 bg-white/5 rounded-2xl shadow-lg opacity-40" />

						<div class="absolute inset-0 w-max h-max flex items-center justify-center gap-6 m-auto">
							<button type="button" class="p-1 rounded-full">
								<SkipBack class="size-6 text-white" />
							</button>

							<Toggle
								loading={false}
								playing={false}
								onClick={() => {}}
								class="bg-white/5"
							/>

							<button type="button" class="p-1 rounded-full">
								<SkipForward class="size-6 text-white" />
							</button>
						</div>
					</div>
				</CircularProgress>
			</div>

			{/* RIGHT: INFO - same tags/classes as <Player /> so line metrics match */}
			<div class="flex-1 min-w-0">
				<small class="text-white text-xs opacity-60 block truncate" aria-hidden>
					<span class="invisible">Album title</span>
				</small>

				<h1 class="text-white text-xl font-semibold truncate">
					<span class="invisible">Song title</span>
				</h1>

				<p class="text-white text-sm opacity-70 truncate" aria-hidden>
					<span class="invisible">Artist name</span>
				</p>

				<div class="mt-3 flex gap-2 justify-between items-center">
					<span class="text-white text-sm opacity-0 tabular-nums shrink-0">
						0:00
					</span>

					<Progress
						value={0}
						max={1}
						min={0}
						class="h-1.5 min-w-0 flex-1 opacity-30"
					/>

					<span class="text-white text-sm opacity-0 tabular-nums shrink-0">
						0:00
					</span>
				</div>

				<div class="flex items-center justify-between gap-4 mt-4 pointer-events-none">
					<Toggle
						loading={false}
						playing={false}
						onClick={() => {}}
						iconClass="size-5"
						class="bg-white/5"
					/>

					<VolumeControl />

					<button type="button" class="rounded-md p-1">
						<ListMusic class="size-6 text-white" />
					</button>
				</div>
			</div>
		</section>
	);
}
