import { ListMusic } from "lucide-preact";

type Props = {
	src?: string;
	alt?: string;
	imgClass?: string;
};

export function PlaylistCover({ src, alt = "", imgClass }: Props) {
	if (src) {
		return (
			<img
				src={src}
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
