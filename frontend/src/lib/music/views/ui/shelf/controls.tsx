import {
	Asterisk,
	GripVertical,
	LayoutGrid,
	Library,
	List,
	Plus,
} from "lucide-preact";
import type { ComponentChildren } from "preact";
import type { SourceFilter, Variant } from "@/lib/music/views/ui/shelf/types";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

interface ControlsProps {
	title?: ComponentChildren;
	toolbar?: ComponentChildren;
	modes?: boolean;
	filterable: boolean;
	sourceFilter: SourceFilter;
	setFilter: (next: SourceFilter) => void;
	variant: Variant;
	setVariant: (next: Variant) => void;
	sortMode?: boolean;
	onSortMode?: () => void;
	onCreate?: () => void;
	showLocal?: boolean;
}

export function Controls({
	title,
	toolbar,
	modes = true,
	filterable,
	sourceFilter,
	setFilter,
	variant,
	setVariant,
	sortMode,
	onSortMode,
	onCreate,
	showLocal = true,
}: ControlsProps) {
	return (
		<div class="relative flex shrink-0 items-center justify-between gap-2 px-3 pt-2 pb-2">
			{title && (
				<span class="pointer-events-none absolute inset-x-28 top-1/2 -translate-y-1/2 truncate text-center text-base font-semibold text-white md:inset-x-32">
					{title}
				</span>
			)}

			<div class="relative flex min-w-0 shrink items-center gap-2">
				{toolbar}
				{filterable && (
					<div class="flex overflow-hidden rounded-md border border-zinc-800 bg-zinc-950">
						<button
							type="button"
							onClick={() => setFilter("all")}
							aria-label={t("music.shelf.all")}
							title={t("music.shelf.all")}
							class={cn(
								"size-10 md:size-8 cursor-pointer flex items-center justify-center transition-colors",
								sourceFilter === "all"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-white",
							)}
						>
							<Asterisk size={18} class="md:hidden" />
							<Asterisk size={14} class="hidden md:block" />
						</button>
						<button
							type="button"
							onClick={() => setFilter("youtube")}
							aria-label={t("music.shelf.source.youtube")}
							class={cn(
								"size-10 md:size-8 cursor-pointer flex items-center justify-center border-l border-zinc-800 transition-colors",
								sourceFilter === "youtube"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-white",
							)}
						>
							<YoutubeIcon class="size-5 md:size-4" />
						</button>
						<button
							type="button"
							onClick={() => setFilter("spotify")}
							aria-label={t("music.shelf.source.spotify")}
							class={cn(
								"size-10 md:size-8 cursor-pointer flex items-center justify-center border-l border-zinc-800 transition-colors",
								sourceFilter === "spotify"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-white",
							)}
						>
							<SpotifyIcon size={18} class="md:hidden" />
							<SpotifyIcon size={14} class="hidden md:block" />
						</button>
						{showLocal && (
							<button
								type="button"
								onClick={() => setFilter("local")}
								aria-label={t("music.custom.tab")}
								class={cn(
									"size-10 md:size-8 cursor-pointer flex items-center justify-center border-l border-zinc-800 transition-colors",
									sourceFilter === "local"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-white",
								)}
							>
								<Library size={18} class="md:hidden" />
								<Library size={14} class="hidden md:block" />
							</button>
						)}
					</div>
				)}

				{onCreate && sourceFilter === "local" && (
					<button
						type="button"
						onClick={onCreate}
						title={t("music.custom.create")}
						aria-label={t("music.custom.create")}
						class="flex size-10 shrink-0 cursor-pointer items-center justify-center rounded-md text-zinc-400 transition-colors hover:text-white md:size-8"
					>
						<Plus size={18} />
					</button>
				)}
			</div>

			<div class="relative flex shrink-0 items-center justify-end gap-2">
				{modes && (
					<div class="flex overflow-hidden rounded-md border border-zinc-800 bg-zinc-950">
						{onSortMode && (
							<button
								type="button"
								onClick={onSortMode}
								aria-pressed={sortMode}
								title={t("music.shelf.sortMode")}
								aria-label={t("music.shelf.sortMode")}
								class={cn(
									"size-10 md:size-8 cursor-pointer flex items-center justify-center transition-colors",
									sortMode
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-white",
								)}
							>
								<GripVertical size={18} class="md:hidden" />
								<GripVertical size={14} class="hidden md:block" />
							</button>
						)}
						<button
							type="button"
							onClick={() => setVariant("grid")}
							aria-label={t("music.shelf.gridView")}
							class={cn(
								"size-10 md:size-8 cursor-pointer flex items-center justify-center transition-colors",
								onSortMode && "border-l border-zinc-800",
								variant === "grid"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-white",
							)}
						>
							<LayoutGrid size={18} class="md:hidden" />
							<LayoutGrid size={14} class="hidden md:block" />
						</button>
						<button
							type="button"
							onClick={() => setVariant("compact")}
							aria-label={t("music.shelf.compactView")}
							class={cn(
								"size-10 md:size-8 cursor-pointer flex items-center justify-center border-l border-zinc-800 transition-colors",
								variant === "compact"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-white",
							)}
						>
							<List size={18} class="md:hidden" />
							<List size={14} class="hidden md:block" />
						</button>
					</div>
				)}
			</div>
		</div>
	);
}
