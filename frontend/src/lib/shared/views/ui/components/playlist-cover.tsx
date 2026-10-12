import { cover } from "@/lib/shared/utils/cover";
import { CoverFallback } from "@/lib/shared/views/ui/components/cover-fallback";
import { Image } from "@/lib/shared/views/ui/components/image";

type Props = {
	src?: string;
	alt?: string;
	imgClass?: string;
	size?: number;
};

export function PlaylistCover({ src, alt = "", imgClass, size = 320 }: Props) {
	return (
		<Image
			src={cover(src, size)}
			alt={alt}
			loading="lazy"
			class={imgClass}
			fallback={<CoverFallback class={imgClass} />}
		/>
	);
}

export default PlaylistCover;
