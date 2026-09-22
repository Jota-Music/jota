import type { ComponentChildren } from "preact";
import { Link } from "wouter-preact";
import { cn } from "@/lib/shared/utils/tw";
import { SourceBadge } from "./badge";
import { Cover } from "./cover";
import { type DragProps, dragFrom, dragOver } from "./hooks/use-reorder";
import type { Item } from "./types";

interface GridProps {
	items: Item[];
	to: (id: string) => string;
	dragProps: (id: string) => DragProps;
	select: (item: Item, e: MouseEvent) => void;
	menu: (item: Item) => (e: MouseEvent) => void;
	actions: (item: Item) => ComponentChildren;
}

export function Grid({
	items,
	to,
	dragProps,
	select,
	menu,
	actions,
}: GridProps) {
	return (
		<div class="grid grid-cols-3 gap-0.5 md:gap-4">
			{items.map((item) => (
				<Link
					key={item.id}
					href={to(item.id)}
					{...dragProps(item.id)}
					onClick={(e) => select(item, e)}
					onContextMenu={menu(item)}
				>
					<div
						class={cn(
							"group flex cursor-pointer select-none flex-col gap-2 overflow-hidden rounded-md p-1.5 transition-shadow duration-150 [-webkit-touch-callout:none] [contain-intrinsic-size:auto_220px] [content-visibility:auto]",
							dragFrom.value === item.id && "opacity-40",
							dragOver.value === item.id &&
								dragFrom.value !== item.id &&
								"bg-(--dominant-color)/10 ring-2 ring-(--dominant-color) ring-inset",
						)}
					>
						<div class="relative aspect-square w-full overflow-hidden rounded-md bg-zinc-900">
							<Cover item={item} size={48} iconSize={28} imageSize={320} />
							<div class="pointer-coarse:opacity-100 absolute right-1 top-1 flex gap-1 opacity-0 transition group-focus-within:opacity-100 group-hover:opacity-100">
								{actions(item)}
							</div>
							{item.source && (
								<div class="absolute bottom-1 left-1 rounded-md bg-black/70 p-0.5">
									<SourceBadge source={item.source} />
								</div>
							)}
						</div>
						<div class="flex flex-col">
							<h3 class="truncate text-sm font-medium text-white">
								{item.name}
							</h3>
							{item.subtitle && (
								<p class="text-xs text-zinc-500">{item.subtitle}</p>
							)}
						</div>
					</div>
				</Link>
			))}
		</div>
	);
}
