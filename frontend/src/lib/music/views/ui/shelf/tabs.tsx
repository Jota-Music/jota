import type { IconType } from "@/lib/music/views/ui/shelf/types";
import { cn } from "@/lib/shared/utils/tw";

export type TabItem<T extends string> = {
	id: T;
	icon: IconType;
	label: string;
};

// The categories a page offers, as one segmented control: a search and a
// channel switch between their tabs the same way, in the shelf's toolbar.
export function Tabs<T extends string>({
	items,
	active,
	onSelect,
}: {
	items: readonly TabItem<T>[];
	active: T;
	onSelect: (id: T) => void;
}) {
	return (
		<div class="flex overflow-hidden rounded-md border border-zinc-800 bg-zinc-950">
			{items.map(({ id, icon: Icon, label }, index) => (
				<button
					key={id}
					type="button"
					title={label}
					aria-label={label}
					aria-pressed={active === id}
					onClick={() => onSelect(id)}
					class={cn(
						"flex size-10 cursor-pointer items-center justify-center transition-colors md:size-8",
						index > 0 && "border-l border-zinc-800",
						active === id
							? "bg-zinc-800 text-white"
							: "text-zinc-500 hover:text-white",
					)}
				>
					<Icon size={16} />
				</button>
			))}
		</div>
	);
}
