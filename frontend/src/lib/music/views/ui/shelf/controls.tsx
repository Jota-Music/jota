import { LayoutGrid, Library, List, Plus } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";
import type { SourceFilter, Variant } from "./types";

interface ControlsProps {
	filterable: boolean;
	sourceFilter: SourceFilter;
	setFilter: (next: SourceFilter) => void;
	variant: Variant;
	setVariant: (next: Variant) => void;
	onCreate?: () => void;
	showLocal?: boolean;
}

export function Controls({
	filterable,
	sourceFilter,
	setFilter,
	variant,
	setVariant,
	onCreate,
	showLocal = true,
}: ControlsProps) {
	return (
		<div class="flex shrink-0 items-center justify-end px-1 pt-1 pb-2 gap-2">
			<div class="mr-auto flex items-center gap-2">
				{filterable && (
					<div class="flex overflow-hidden rounded-md border border-zinc-800">
						<button
							type="button"
							onClick={() => setFilter("all")}
							aria-label={t("music.shelf.all")}
							class={cn(
								"h-10 md:h-8 px-3 cursor-pointer flex items-center justify-center transition-colors text-xs font-medium",
								sourceFilter === "all"
									? "bg-zinc-800 text-white"
									: "text-zinc-500 hover:text-white",
							)}
						>
							All
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
						class="flex size-10 shrink-0 cursor-pointer items-center justify-center rounded-md border border-zinc-800 text-zinc-400 transition-colors hover:text-white md:size-8"
					>
						<Plus size={20} class="md:hidden" />
						<Plus size={16} class="hidden md:block" />
					</button>
				)}
			</div>
			<div class="flex overflow-hidden rounded-md border border-zinc-800">
				<button
					type="button"
					onClick={() => setVariant("grid")}
					aria-label={t("music.shelf.gridView")}
					class={cn(
						"size-10 md:size-8 cursor-pointer flex items-center justify-center transition-colors",
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
		</div>
	);
}
