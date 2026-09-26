import { Pause, Play, Plus, Trash2 } from "lucide-preact";
import { useRef } from "preact/hooks";
import { spotifyConnected } from "@/lib/auth/views/stores/session";
import { SpotifyConnect } from "@/lib/auth/views/ui/spotify-connect";
import { isPlaying } from "@/lib/music/views/stores/audio";
import { queueSource } from "@/lib/music/views/stores/queue";
import { t } from "@/lib/shared/i18n";
import {
	type Action,
	openContextMenu,
} from "@/lib/shared/views/ui/components/context-menu";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";
import { Compact } from "./compact";
import { Controls } from "./controls";
import { Placeholder } from "./cover";
import { Grid } from "./grid";
import { LocalHint } from "./hint";
import { useReorder } from "./hooks/use-reorder";
import { defaultViewKey, useView } from "./hooks/use-view";
import type { Item, Props, Variant } from "./types";

export { FollowingHint, LocalHint, RelayHint, YouTubeHint } from "./hint";
export type { IconType, Item, Source } from "./types";

const tiles = Array.from({ length: 9 }, (_, i) => i);

// Reserves the space the loaded list will take, so the shelf does not jump
// when the data lands.
function Loading({ variant }: { variant: Variant }) {
	return (
		<div class="min-h-0 flex-1 overflow-hidden">
			{variant === "grid" ? (
				<div class="grid grid-cols-3 gap-0.5 px-1 pt-1 md:gap-4">
					{tiles.map((i) => (
						<div
							key={i}
							class="flex aspect-square w-full animate-pulse items-center justify-center rounded-md bg-zinc-900 text-zinc-700"
						>
							<Placeholder size={28} />
						</div>
					))}
				</div>
			) : (
				<ul class="flex flex-col">
					{tiles.map((i) => (
						<li
							key={i}
							class="flex animate-pulse items-center gap-3 px-2 py-1.5"
						>
							<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded bg-zinc-900 text-zinc-700">
								<Placeholder size={14} />
							</div>
							<div class="h-3 w-1/3 rounded bg-zinc-900" />
						</li>
					))}
				</ul>
			)}
		</div>
	);
}

export function Shelf({
	items,
	to,
	viewKey = defaultViewKey,
	onSelect,
	onPlay,
	actions,
	leading,
	isLoading = false,
	emptyMessage = t("music.shelf.empty"),
	onRemove,
	onReorder,
	onCreate,
	showLocal = true,
}: Props) {
	const {
		variant,
		filter: sourceFilter,
		setVariant,
		setFilter,
	} = useView(viewKey);
	const listRef = useRef<HTMLDivElement>(null);
	const { dragProps, captureClick } = useReorder({
		count: items.length,
		listRef,
		onReorder,
	});

	// A right-click (or touch long-press) opens the item's own actions instead
	// of the browser context menu.
	const openItemMenu = (item: Item) => (e: MouseEvent) => {
		const actions: Action[] = [];
		if (onPlay) {
			const playing = queueSource.value === item.id && isPlaying.value;
			actions.push({
				icon: playing ? Pause : Play,
				label: playing ? t("music.pause") : t("music.play"),
				run: () => onPlay(item.id),
			});
		}
		actions.push(...(item.menu ?? []));
		if (onRemove && item.removable) {
			actions.push({
				icon: Trash2,
				label: t("common.remove"),
				danger: true,
				run: () => onRemove(item.id),
			});
		}
		openContextMenu(actions, e);
	};

	const renderActions = (item: Item) => actions?.(item.id) ?? null;

	const select = (item: Item, e: MouseEvent) => {
		if (!onSelect) return;
		e.preventDefault();
		onSelect(item.id);
	};

	// The left edge carries the destructive and secondary toggles, keeping the
	// right edge for playing whatever the item is.
	const renderLeading = (item: Item) => {
		const extra = leading?.(item.id);
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
						<Trash2 size={16} />
					</button>
				)}
			</>
		);
	};

	const hoverable = (item: Item) =>
		(onRemove && item.removable) || !!actions || !!leading;

	const filterable = items.some((item) => item.source);

	const filteredItems =
		sourceFilter === "all" || !filterable
			? items
			: items.filter((i) => i.source === sourceFilter);

	return (
		<div class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border border-zinc-800 bg-zinc-950">
			{isLoading ? (
				<Loading variant={variant} />
			) : items.length === 0 ? (
				<div class="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-sm text-zinc-500">
					{emptyMessage}
					{onCreate && (
						<button
							type="button"
							onClick={onCreate}
							title={t("music.custom.create")}
							aria-label={t("music.custom.create")}
							class="flex h-10 md:h-8 cursor-pointer items-center gap-2 rounded-md border border-zinc-800 px-4 text-zinc-400 transition-colors hover:text-white"
						>
							<Plus size={18} class="md:hidden" />
							<Plus size={14} class="hidden md:block" />
							{t("music.custom.create")}
						</button>
					)}
				</div>
			) : (
				<>
					<Controls
						filterable={filterable}
						sourceFilter={sourceFilter}
						setFilter={setFilter}
						variant={variant}
						setVariant={setVariant}
						onCreate={onCreate}
						showLocal={showLocal}
					/>

					<div class="relative min-h-0 flex-1">
						<div
							ref={listRef}
							class="h-full overflow-y-auto px-1 pb-3"
							onClickCapture={captureClick}
						>
							{filteredItems.length === 0 ? (
								<div class="flex h-full items-center justify-center p-8 text-sm text-zinc-500">
									{sourceFilter === "spotify" && !spotifyConnected.value ? (
										<SpotifyConnect />
									) : sourceFilter === "spotify" ? (
										t("music.shelf.noSpotify")
									) : sourceFilter === "local" ? (
										<LocalHint />
									) : (
										emptyMessage
									)}
								</div>
							) : variant === "grid" ? (
								<Grid
									items={filteredItems}
									to={to}
									dragProps={dragProps}
									select={select}
									menu={openItemMenu}
									actions={renderActions}
									leading={renderLeading}
								/>
							) : (
								<Compact
									items={filteredItems}
									to={to}
									dragProps={dragProps}
									select={select}
									menu={openItemMenu}
									actions={renderActions}
									leading={renderLeading}
									hoverable={hoverable}
								/>
							)}
						</div>
						<Scrollbar target={listRef} />
					</div>
				</>
			)}
		</div>
	);
}
