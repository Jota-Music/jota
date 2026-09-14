import { Disc3, LayoutGrid, List, X } from "lucide-preact";
import { useState } from "preact/hooks";
import { Link } from "wouter-preact";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

const storageKey = "cover_grid_view";
const filterKey = "playlist_source_filter";

type Variant = "grid" | "compact";
type SourceFilter = "all" | "spotify" | "youtube";

export type Source = "spotify" | "youtube";

function loadVariant(): Variant {
	return localStorage.getItem(storageKey) === "compact" ? "compact" : "grid";
}

function loadFilter(): SourceFilter {
	return (localStorage.getItem(filterKey) as SourceFilter) ?? "all";
}

function SourceBadge({ source }: { source?: Source }) {
	if (!source) return null;
	return (
		<span
			class="flex size-5 items-center justify-center text-zinc-400"
			title={source === "spotify" ? "Spotify" : "YouTube"}
		>
			{source === "spotify" ? (
				<SpotifyIcon size={14} />
			) : (
				<YoutubeIcon class="size-4" />
			)}
		</span>
	);
}

export interface Item {
	id: string;
	name: string;
	cover?: string;
	subtitle?: string;
	source?: Source;
	removable?: boolean;
}

interface Props {
	items: Item[];
	to: (id: string) => string;
	isLoading?: boolean;
	emptyMessage?: string;
	onRemove?: (id: string) => void;
}

export function Shelf({
	items,
	to,
	isLoading = false,
	emptyMessage = "No items found.",
	onRemove,
}: Props) {
	const [variant, setVariant] = useState<Variant>(loadVariant);
	const [sourceFilter, setSourceFilter] = useState<SourceFilter>(loadFilter);

	const toggle = (next: Variant) => {
		setVariant(next);
		localStorage.setItem(storageKey, next);
	};

	const setFilter = (next: SourceFilter) => {
		setSourceFilter(next);
		localStorage.setItem(filterKey, next);
	};

	const filteredItems =
		sourceFilter === "all"
			? items
			: items.filter((i) => i.source === sourceFilter);

	return (
		<div class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border border-zinc-800 bg-zinc-950">
			{isLoading ? (
				<div class="flex flex-1 items-center justify-center p-8 text-sm text-zinc-400">
					Loading...
				</div>
			) : items.length === 0 ? (
				<div class="flex flex-1 items-center justify-center p-8 text-sm text-zinc-500">
					{emptyMessage}
				</div>
			) : (
				<>
					<div class="flex shrink-0 justify-between px-3 py-2 gap-2">
						<div class="flex overflow-hidden rounded-md border border-zinc-800">
							<button
								type="button"
								onClick={() => setFilter("all")}
								aria-label="All"
								class={`flex h-8 px-2 gap-1 cursor-pointer items-center justify-center transition-colors text-xs font-medium ${
									sourceFilter === "all"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-white"
								}`}
							>
								All
							</button>
							<button
								type="button"
								onClick={() => setFilter("spotify")}
								aria-label="Spotify"
								class={`flex h-8 w-8 cursor-pointer items-center justify-center border-l border-zinc-800 transition-colors ${
									sourceFilter === "spotify"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-white"
								}`}
							>
								<SpotifyIcon size={14} />
							</button>
							<button
								type="button"
								onClick={() => setFilter("youtube")}
								aria-label="YouTube"
								class={`flex h-8 w-8 cursor-pointer items-center justify-center border-l border-zinc-800 transition-colors ${
									sourceFilter === "youtube"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-white"
								}`}
							>
								<YoutubeIcon class="size-4" />
							</button>
						</div>
						<div class="flex overflow-hidden rounded-md border border-zinc-800">
							<button
								type="button"
								onClick={() => toggle("grid")}
								aria-label="Vista grilla"
								class={`flex h-8 w-8 cursor-pointer items-center justify-center transition-colors ${
									variant === "grid"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-white"
								}`}
							>
								<LayoutGrid size={14} />
							</button>
							<button
								type="button"
								onClick={() => toggle("compact")}
								aria-label="Vista compacta"
								class={`flex h-8 w-8 cursor-pointer items-center justify-center border-l border-zinc-800 transition-colors ${
									variant === "compact"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-white"
								}`}
							>
								<List size={14} />
							</button>
						</div>
					</div>

					<div class="min-h-0 flex-1 overflow-y-auto px-3 pb-3">
						{filteredItems.length === 0 ? (
							<div class="flex h-full items-center justify-center p-8 text-sm text-zinc-500">
								{sourceFilter === "spotify"
									? "No Spotify playlists."
									: "No YouTube playlists."}
							</div>
						) : variant === "grid" ? (
							<div class="grid grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
								{filteredItems.map((item) => (
									<Link key={item.id} href={to(item.id)}>
										<div class="group flex cursor-pointer flex-col gap-2 overflow-hidden rounded-md">
											<div class="relative aspect-square w-full overflow-hidden rounded-md bg-zinc-900">
												{item.cover ? (
													<img
														src={item.cover}
														alt={item.name}
														loading="lazy"
														decoding="async"
														class="h-full w-full object-cover transition-opacity group-hover:opacity-80"
													/>
												) : (
													<div class="flex h-full w-full items-center justify-center text-zinc-600">
														<Disc3 size={28} />
													</div>
												)}
												{onRemove && item.removable && (
													<button
														type="button"
														title="Quitar"
														onClick={(e) => {
															e.preventDefault();
															e.stopPropagation();
															onRemove(item.id);
														}}
														class="absolute right-1 top-1 flex h-6 w-6 cursor-pointer items-center justify-center rounded-md bg-black/70 text-zinc-300 opacity-0 transition group-hover:opacity-100 hover:text-white"
													>
														<X size={14} />
													</button>
												)}
												{item.source && (
													<div class="absolute bottom-1 left-1 rounded-md bg-black/70 p-0.5">
														<SourceBadge source={item.source} />
													</div>
												)}
											</div>
											<div class="flex flex-col">
												<h3 class="truncate text-sm font-medium text-zinc-300 transition-colors group-hover:text-white">
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
						) : (
							<ul class="flex flex-col">
								{filteredItems.map((item) => (
									<li key={item.id}>
										<Link href={to(item.id)}>
											<div class="group relative flex cursor-pointer items-center gap-3 rounded-md px-2 py-1.5 hover:bg-zinc-900">
												<div class="h-10 w-10 shrink-0 overflow-hidden rounded bg-zinc-900">
													{item.cover ? (
														<img
															src={item.cover}
															alt={item.name}
															loading="lazy"
															decoding="async"
															class="h-full w-full object-cover transition-opacity group-hover:opacity-80"
														/>
													) : (
														<div class="flex h-full w-full items-center justify-center text-zinc-600">
															<Disc3 size={14} />
														</div>
													)}
												</div>
												<div class="flex min-w-0 flex-1 flex-col">
													<h3 class="truncate text-sm text-zinc-300 group-hover:text-white">
														{item.name}
													</h3>
													{item.subtitle && (
														<p class="truncate text-xs text-zinc-500">
															{item.subtitle}
														</p>
													)}
												</div>
												{item.source && (
													<span
														class={
															onRemove && item.removable
																? "shrink-0 transition-opacity group-hover:opacity-0"
																: "shrink-0"
														}
													>
														<SourceBadge source={item.source} />
													</span>
												)}
												{onRemove && item.removable && (
													<button
														type="button"
														title="Quitar"
														onClick={(e) => {
															e.preventDefault();
															e.stopPropagation();
															onRemove(item.id);
														}}
														class="absolute right-2 top-1/2 flex h-6 w-6 -translate-y-1/2 cursor-pointer items-center justify-center rounded-md text-zinc-500 opacity-0 transition group-hover:opacity-100 hover:bg-zinc-800 hover:text-white"
													>
														<X size={14} />
													</button>
												)}
											</div>
										</Link>
									</li>
								))}
							</ul>
						)}
					</div>
				</>
			)}
		</div>
	);
}
