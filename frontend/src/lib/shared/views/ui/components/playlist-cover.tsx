import { ListMusic } from "lucide-preact";
import { cover } from "@/lib/shared/utils/cover";

type Props = {
	src?: string;
	alt?: string;
	imgClass?: string;
	size?: number;
};

export function PlaylistCover({ src, alt = "", imgClass, size = 320 }: Props) {
	if (src) {
		return (
			<img
				src={cover(src, size)}
				alt={alt}
				loading="lazy"
				decoding="async"
				class={imgClass}
			/>
		);
	}
	return (
		<div class="flex h-full w-full items-center justify-center bg-zinc-900 text-zinc-600">
			<ListMusic size={28} />
		</div>
	);
}

export default PlaylistCover;
