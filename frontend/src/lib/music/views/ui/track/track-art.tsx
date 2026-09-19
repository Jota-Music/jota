import { Loader, Music, Pause } from "lucide-preact";
import type { Song } from "@/lib/music/model";
import { cn } from "@/lib/shared/utils/tw";

type Props = {
	song: Song;
	current?: boolean;
	loading?: boolean;
	playing?: boolean;
	alt?: string;
	class?: string;
	imgClass?: string;
};

export function TrackArt({
	song,
	current = false,
	loading = false,
	playing = false,
	alt = "",
	class: className,
	imgClass,
}: Props) {
	return (
		<div class={cn("relative", className)}>
			<img
				loading="lazy"
				decoding="async"
				src={song.album?.covers?.[0]}
				alt={alt}
				class={cn(imgClass, current && "brightness-40")}
			/>

			{current &&
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
		</div>
	);
}

export default TrackArt;
