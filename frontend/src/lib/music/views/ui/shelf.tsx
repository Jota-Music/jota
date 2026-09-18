import { signal } from "@preact/signals";
import { Disc3, LayoutGrid, Library, List, Plus, X } from "lucide-preact";
import type { ComponentChildren, ComponentType } from "preact";
import { useRef, useState } from "preact/hooks";
import { Link } from "wouter-preact";
import { spotifyConnected } from "@/lib/auth/views/stores/session";
import { SpotifyConnect } from "@/lib/auth/views/ui/spotify-connect";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";
import { usePointerDrag } from "@/lib/shared/views/ui/hooks/use-pointer-drag";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

const defaultViewKey = "cover_grid_view";
const filterKey = "playlist_source_filter";
const COARSE = window.matchMedia("(pointer: coarse)").matches;

const dragFrom = signal<string | null>(null);
const dragOver = signal<string | null>(null);

type Variant = "grid" | "compact";
type SourceFilter = "all" | "spotify" | "youtube" | "local";

export type Source = "spotify" | "youtube" | "local";

function loadVariant(key: string): Variant {
	return localStorage.getItem(key) === "compact" ? "compact" : "grid";
}

function loadFilter(): SourceFilter {
	const saved = localStorage.getItem(filterKey);
	return saved === "spotify" || saved === "youtube" || saved === "local"
		? saved
		: "all";
}

export type IconType = ComponentType<{ size?: number | string }>;

function Placeholder({
	icon: Icon = Disc3,
	size,
}: {
	icon?: IconType;
	size: number;
}) {
	return <Icon size={size} />;
}

function SourceBadge({ source }: { source?: Source }) {
	if (!source) return null;
	const title =
		source === "spotify"
			? t("music.shelf.source.spotify")
			: source === "youtube"
				? t("music.shelf.source.youtube")
				: t("music.custom.tab");
	return (
		<span
			class="flex size-5 items-center justify-center text-zinc-400"
			title={title}
		>
			{source === "spotify" ? (
				<SpotifyIcon size={14} />
			) : source === "youtube" ? (
				<YoutubeIcon class="size-4" />
			) : (
				<Library size={14} />
			)}
		</span>
	);
}

export function YouTubeHint() {
	const hintSteps = [
		t("music.hint.step1"),
		t("music.hint.step2"),
		t("music.hint.step3"),
	];

	return (
		<div class="relative flex w-full max-w-sm flex-col items-center gap-5 px-6 py-8 text-center">
			<div class="pointer-events-none absolute top-0 size-32 rounded-full bg-red-600/20 blur-3xl" />

			<div class="relative flex size-12 items-center justify-center rounded-full bg-red-500/10 ring-1 ring-red-500/25 ring-inset">
				<YoutubeIcon class="size-6 text-red-500" />
			</div>

			<div class="relative flex flex-col gap-1">
				<p class="text-sm font-semibold text-zinc-100">
					{t("music.hint.title")}
				</p>
				<p class="text-xs text-zinc-500">{t("music.hint.body")}</p>
			</div>

			<ol class="relative flex w-full flex-col text-left">
				{hintSteps.map((step, index) => (
					<li key={step} class="relative flex gap-3 pb-4 last:pb-0">
						{index < hintSteps.length - 1 && (
							<span class="absolute left-3 top-7 h-[calc(100%-1.75rem)] w-px bg-zinc-800" />
						)}
						<span class="relative z-10 flex size-6 shrink-0 items-center justify-center rounded-full border border-zinc-700 bg-zinc-900 text-[11px] font-semibold text-zinc-300 tabular-nums">
							<span class="translate-y-[0.06em] leading-none">{index + 1}</span>
						</span>
						<span class="pt-0.5 text-xs leading-snug text-zinc-400">
							{step}
						</span>
					</li>
				))}
			</ol>
		</div>
	);
}

export interface Item {
	id: string;
	name: string;
	cover?: string;
	subtitle?: ComponentChildren;
	source?: Source;
	removable?: boolean;
	icon?: IconType;
	// placeholder fills the cover slot when there is no cover, taking over from
	// `icon`. It receives the size the slot expects.
	placeholder?: (size: number) => ComponentChildren;
}

interface Props {
	items: Item[];
	to: (id: string) => string;
	// viewKey scopes the grid/compact choice to one shelf, so each can keep its
	// own layout.
	viewKey?: string;
	onSelect?: (id: string) => void;
	// actions renders hover buttons for an item, next to the remove button.
	actions?: (id: string) => ComponentChildren;
	isLoading?: boolean;
	emptyMessage?: ComponentChildren;
	onRemove?: (id: string) => void;
	onReorder?: (fromId: string, toId: string) => void;
	// onCreate renders a "new playlist" button next to the source filters.
	onCreate?: () => void;
}

export function Shelf({
	items,
	to,
	viewKey = defaultViewKey,
	onSelect,
	actions,
	isLoading = false,
	emptyMessage = t("music.shelf.empty"),
	onRemove,
	onReorder,
	onCreate,
}: Props) {
	const [variant, setVariant] = useState<Variant>(() => loadVariant(viewKey));
	const [sourceFilter, setSourceFilter] = useState<SourceFilter>(loadFilter);
	const listRef = useRef<HTMLDivElement>(null);
	const reorderable = !!onReorder && items.length > 1;

	const endDrag = () => {
		dragFrom.value = null;
		dragOver.value = null;
	};

	const startDrag = (id: string, e: DragEvent) => {
		if ((e.target as Element | null)?.closest("button")) {
			e.preventDefault();
			return;
		}
		dragFrom.value = id;
		const dt = e.dataTransfer;
		if (!dt) return;
		try {
			dt.setData("text/plain", id);
			dt.effectAllowed = "move";
		} catch {
			/* noop */
		}
	};

	const overItem = (id: string, e: DragEvent) => {
		if (dragFrom.value == null) return;
		e.preventDefault();
		const dt = e.dataTransfer;
		if (dt) {
			try {
				dt.dropEffect = "move";
			} catch {
				/* noop */
			}
		}
		if (dragOver.value !== id) dragOver.value = id;
	};

	const dropItem = (id: string, e: DragEvent) => {
		const from = dragFrom.value;
		if (from == null) return;
		e.preventDefault();
		endDrag();
		if (from !== id) onReorder?.(from, id);
	};

	const source = useRef<string | null>(null);

	const { start, captureClick } = usePointerDrag({
		begin: () => {
			if (source.current == null) return;
			dragFrom.value = source.current;
			dragOver.value = source.current;
		},
		move: (e) => {
			const el = document.elementFromPoint(e.clientX, e.clientY);
			const id = el
				?.closest("[data-reorder-id]")
				?.getAttribute("data-reorder-id");
			if (id != null && dragOver.value !== id) dragOver.value = id;
		},
		end: (dragged) => {
			const from = dragFrom.value;
			const to = dragOver.value;
			endDrag();
			if (dragged && from != null && to != null && from !== to) {
				onReorder?.(from, to);
			}
		},
	});

	const handlePointerDown = (id: string) => (e: PointerEvent) => {
		if (!reorderable) return;
		if ((e.target as Element).closest("button")) return;
		if (!start(e)) return;
		source.current = id;
	};

	// A touch long-press starts the drag, but the browser would rather open the
	// link menu. Suppress it on coarse pointers; keep the desktop context menu.
	const preventMenu = (e: MouseEvent) => {
		if (reorderable && COARSE) e.preventDefault();
	};

	const renderActions = (item: Item) => {
		const extra = actions?.(item.id);
		const removable = onRemove && item.removable;
		if (!extra && !removable) return null;
		return (
			<>
				{extra}
				{removable && (
					<button
						type="button"
						title={t("common.remove")}
						onClick={(e) => {
							e.preventDefault();
							e.stopPropagation();
							onRemove?.(item.id);
						}}
						class="flex h-8 w-8 cursor-pointer items-center justify-center rounded-md bg-black/70 text-zinc-300 hover:text-white"
					>
						<X size={16} />
					</button>
				)}
			</>
		);
	};

	const select = (item: Item, e: MouseEvent) => {
		if (!onSelect) return;
		e.preventDefault();
		onSelect(item.id);
	};

	const toggle = (next: Variant) => {
		setVariant(next);
		localStorage.setItem(viewKey, next);
	};

	const setFilter = (next: SourceFilter) => {
		setSourceFilter(next);
		localStorage.setItem(filterKey, next);
	};

	const filterable = items.some((item) => item.source);

	const filteredItems =
		sourceFilter === "all" || !filterable
			? items
			: items.filter((i) => i.source === sourceFilter);

	return (
		<div class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border border-zinc-800 bg-zinc-950">
			{isLoading ? (
				<div class="flex flex-1 items-center justify-center p-8 text-sm text-zinc-400">
					{t("common.loading")}
				</div>
			) : items.length === 0 ? (
				<div class="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-sm text-zinc-500">
					{emptyMessage}
					{onCreate && (
						<button
							type="button"
							onClick={onCreate}
							title={t("music.custom.create")}
							aria-label={t("music.custom.create")}
							class="flex h-10 cursor-pointer items-center gap-2 rounded-md border border-zinc-800 px-4 text-zinc-400 transition-colors hover:text-white"
						>
							<Plus size={18} />
							{t("music.custom.create")}
						</button>
					)}
				</div>
			) : (
				<>
					<div class="flex shrink-0 items-center justify-end px-3 py-2 gap-2">
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
									<Plus size={18} class="md:hidden" />
									<Plus size={14} class="hidden md:block" />
								</button>
							)}
						</div>
						<div class="flex overflow-hidden rounded-md border border-zinc-800">
							<button
								type="button"
								onClick={() => toggle("grid")}
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
								onClick={() => toggle("compact")}
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

					<div class="relative min-h-0 flex-1">
						<div
							ref={listRef}
							class="h-full overflow-y-auto px-3 pb-3"
							onClickCapture={captureClick}
						>
							{filteredItems.length === 0 ? (
								<div class="flex h-full items-center justify-center p-8 text-sm text-zinc-500">
									{sourceFilter === "spotify" && !spotifyConnected.value ? (
										<SpotifyConnect />
									) : sourceFilter === "spotify" ? (
										t("music.shelf.noSpotify")
									) : sourceFilter === "local" ? (
										t("music.custom.empty")
									) : (
										<YouTubeHint />
									)}
								</div>
							) : variant === "grid" ? (
								<div class="grid grid-cols-3 gap-0.5 md:gap-4">
									{filteredItems.map((item) => (
										<Link
											key={item.id}
											href={to(item.id)}
											data-reorder-id={item.id}
											draggable={reorderable && COARSE}
											onPointerDown={handlePointerDown(item.id)}
											onClick={(e) => select(item, e)}
											onContextMenu={preventMenu}
											onDragStart={(e) => startDrag(item.id, e)}
											onDragOver={(e) => overItem(item.id, e)}
											onDrop={(e) => dropItem(item.id, e)}
											onDragEnd={endDrag}
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
													{item.cover ? (
														<img
															src={item.cover}
															alt={item.name}
															draggable={false}
															loading="lazy"
															decoding="async"
															class="h-full w-full object-cover transition-opacity group-hover:opacity-80"
														/>
													) : (
														<div class="flex h-full w-full items-center justify-center text-zinc-600">
															{item.placeholder?.(48) ?? (
																<Placeholder icon={item.icon} size={28} />
															)}
														</div>
													)}
													<div class="pointer-coarse:opacity-100 absolute right-1 top-1 flex gap-1 opacity-0 transition group-focus-within:opacity-100 group-hover:opacity-100">
														{renderActions(item)}
													</div>
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
										<li
											key={item.id}
											class="[contain-intrinsic-size:auto_52px] [content-visibility:auto]"
										>
											<Link
												href={to(item.id)}
												data-reorder-id={item.id}
												draggable={reorderable && COARSE}
												onPointerDown={handlePointerDown(item.id)}
												onClick={(e) => select(item, e)}
												onContextMenu={preventMenu}
												onDragStart={(e) => startDrag(item.id, e)}
												onDragOver={(e) => overItem(item.id, e)}
												onDrop={(e) => dropItem(item.id, e)}
												onDragEnd={endDrag}
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
													<div class="h-10 w-10 shrink-0 overflow-hidden rounded bg-zinc-900">
														{item.cover ? (
															<img
																src={item.cover}
																alt={item.name}
																draggable={false}
																loading="lazy"
																decoding="async"
																class="h-full w-full object-cover transition-opacity group-hover:opacity-80"
															/>
														) : (
															<div class="flex h-full w-full items-center justify-center text-zinc-600">
																{item.placeholder?.(22) ?? (
																	<Placeholder icon={item.icon} size={14} />
																)}
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
																(onRemove && item.removable) || actions
																	? "shrink-0 transition-opacity group-hover:opacity-0"
																	: "shrink-0"
															}
														>
															<SourceBadge source={item.source} />
														</span>
													)}
													<div class="pointer-coarse:opacity-100 absolute right-2 top-1/2 flex -translate-y-1/2 gap-1 opacity-0 transition group-focus-within:opacity-100 group-hover:opacity-100">
														{renderActions(item)}
													</div>
												</div>
											</Link>
										</li>
									))}
								</ul>
							)}
						</div>
						<Scrollbar target={listRef} />
					</div>
				</>
			)}
		</div>
	);
}
