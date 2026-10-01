import { Disc3 } from "lucide-preact";
import type { IconType, Item } from "@/lib/music/views/ui/shelf/types";
import { cover } from "@/lib/shared/utils/cover";
import { Image } from "@/lib/shared/views/ui/components/image";
import { Mosaic } from "@/lib/shared/views/ui/components/mosaic";

export function Placeholder({
	icon: Icon = Disc3,
	size,
}: {
	icon?: IconType;
	size: number;
}) {
	return <Icon size={size} />;
}

export function Cover({
	item,
	size,
	iconSize,
	imageSize = 480,
}: {
	item: Item;
	size: number;
	iconSize: number;
	imageSize?: number;
}) {
	const fallback = item.placeholder?.(size) ?? (
		<Placeholder icon={item.icon} size={iconSize} />
	);

	return (
		<>
			{item.covers?.length ? (
				<Mosaic
					covers={item.covers}
					alt={item.name}
					size={imageSize}
					placeholder={fallback}
				/>
			) : (
				<Image
					src={cover(item.cover, imageSize)}
					alt={item.name}
					draggable={false}
					loading="lazy"
					class="h-full w-full object-cover opacity-80"
					fallback={
						<div class="flex h-full w-full items-center justify-center text-zinc-600">
							{fallback}
						</div>
					}
				/>
			)}
			{item.overlay && (
				<div class="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/60">
					{item.overlay(size)}
				</div>
			)}
		</>
	);
}
