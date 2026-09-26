import type { ComponentChildren } from "preact";
import { Link } from "wouter-preact";
import { cn } from "@/lib/shared/utils/tw";
import { hover } from "../cover-hover";
import { SourceBadge } from "./badge";
import { Cover } from "./cover";
import { type DragProps, dragFrom, dragOver } from "./hooks/use-reorder";
import type { Item } from "./types";

interface CompactProps {
	items: Item[];
	to: (id: string) => string;
	dragProps: (id: string) => DragProps;
	select: (item: Item, e: MouseEvent) => void;
	menu: (item: Item) => (e: MouseEvent) => void;
	actions: (item: Item) => ComponentChildren;
	leading: (item: Item) => ComponentChildren;
	// hoverable fades the source badge out on hover to reveal the actions.
	hoverable: (item: Item) => boolean;
}

export function Compact({
	items,
	to,
	dragProps,
	select,
	menu,
	actions,
	leading,
	hoverable,
}: CompactProps) {
	return (
		<ul class="flex flex-col">
			{items.map((item) => (
				<li
					key={item.id}
					class="[contain-intrinsic-size:auto_52px] [content-visibility:auto]"
				>
					<Link
						href={to(item.id)}
						{...dragProps(item.id)}
						onClick={(e) => select(item, e)}
						onContextMenu={menu(item)}
					>
						<div
							class={cn(
								"group relative flex cursor-pointer select-none items-center gap-3 rounded-md px-2 py-1.5 transition-shadow duration-150 [-webkit-touch-callout:none] hover:bg-zinc-900",
								dragFrom.value === item.id && "opacity-40",
								dragOver.value === item.id &&
									dragFrom.value !== item.id &&
									"bg-(--dominant-color)/15 ring-1 ring-(--dominant-color)/50 ring-inset",
							)}
						>
							<div class="relative h-10 w-10 shrink-0 overflow-hidden rounded bg-zinc-900">
								<Cover item={item} size={22} iconSize={14} imageSize={80} />
							</div>
							<div class="flex min-w-0 flex-1 flex-col">
								<h3 class="truncate text-sm text-white">{item.name}</h3>
								{item.subtitle && (
									<p class="truncate text-xs text-zinc-500">{item.subtitle}</p>
								)}
							</div>
							{item.source && (
								<span
									class={cn(
										"shrink-0",
										hoverable(item) &&
											"transition-opacity group-hover:opacity-0",
									)}
								>
									<SourceBadge source={item.source} />
								</span>
							)}
							<div class={cn(hover, "right-2 top-1/2 -translate-y-1/2")}>
								{/* The thumbnail is too small for its own corner button, so
								    leading joins the right cluster here. */}
								{leading(item)}
								{actions(item)}
							</div>
						</div>
					</Link>
				</li>
			))}
		</ul>
	);
}
