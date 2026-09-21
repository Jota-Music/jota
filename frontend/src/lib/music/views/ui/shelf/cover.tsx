import { Disc3 } from "lucide-preact";
import { cover } from "@/lib/shared/utils/cover";
import { Mosaic } from "@/lib/shared/views/ui/components/mosaic";
import type { IconType, Item } from "./types";

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
	return (
		<>
			{item.covers?.length ? (
				<Mosaic
					covers={item.covers}
					alt={item.name}
					size={imageSize}
					placeholder={
						item.placeholder?.(size) ?? (
							<Placeholder icon={item.icon} size={iconSize} />
						)
					}
				/>
			) : item.cover ? (
				<img
					src={cover(item.cover, imageSize)}
					alt={item.name}
					draggable={false}
					loading="lazy"
					decoding="async"
					class="h-full w-full object-cover opacity-80"
				/>
			) : (
				<div class="flex h-full w-full items-center justify-center text-zinc-600">
					{item.placeholder?.(size) ?? (
						<Placeholder icon={item.icon} size={iconSize} />
					)}
				</div>
			)}
			{item.overlay && (
				<div class="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/60">
					{item.overlay(size)}
				</div>
			)}
		</>
	);
}
