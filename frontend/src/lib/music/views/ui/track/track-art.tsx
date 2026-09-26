import { Loader, Music, Pause } from "lucide-preact";
import type { Song } from "@/lib/music/model";
import { cover } from "@/lib/shared/utils/cover";
import { cn } from "@/lib/shared/utils/tw";
import { Image } from "@/lib/shared/views/ui/components/image";

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
			<Image
				loading="lazy"
				decoding="async"
				src={cover(song.album?.covers?.[0], 80)}
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
						// Lucide strokes in currentColor, so the fill has to match the
						// text color or the stroke shows as a light outline around the bars.
						class="absolute inset-0 z-10 m-auto text-(--dominant-color) fill-current drop-shadow-md drop-shadow-black"
					/>
				))}
		</div>
	);
}

export default TrackArt;
