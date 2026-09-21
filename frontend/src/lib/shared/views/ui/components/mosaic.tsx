import type { ComponentChildren } from "preact";
import { cover } from "@/lib/shared/utils/cover";
import { cn } from "@/lib/shared/utils/tw";

type Props = {
	covers: string[];
	alt?: string;
	placeholder?: ComponentChildren;
	size?: number;
};

export function Mosaic({ covers, alt = "", placeholder, size = 480 }: Props) {
	const imgs = covers.slice(0, 4);

	if (imgs.length === 0) return null;

	if (imgs.length === 1) {
		return (
			<img
				src={cover(imgs[0], size)}
				alt={alt}
				loading="lazy"
				decoding="async"
				draggable={false}
				class="h-full w-full object-cover"
			/>
		);
	}

	// A quadrant only fills a quarter of the tile, so half the resolution is
	// more than enough and saves four full-size downloads per playlist.
	const tile = Math.ceil(size / 2);

	return (
		<div
			class={cn(
				"grid h-full w-full grid-cols-2",
				imgs.length > 2 && "grid-rows-2",
			)}
		>
			{imgs.map((src) => (
				<img
					key={src}
					src={cover(src, tile)}
					alt={alt}
					loading="lazy"
					decoding="async"
					draggable={false}
					class="h-full w-full object-cover"
				/>
			))}
			{imgs.length === 3 && (
				<div class="flex h-full w-full items-center justify-center text-zinc-600">
					{placeholder}
				</div>
			)}
		</div>
	);
}

export default Mosaic;
