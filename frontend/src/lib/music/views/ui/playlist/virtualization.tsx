import { computed, signal } from "@preact/signals";
import {
	ChevronDown,
	ChevronUp,
	GripVertical,
	TriangleAlert,
} from "lucide-preact";
import { memo } from "preact/compat";
import { useEffect, useLayoutEffect, useRef } from "preact/hooks";
import { useLocation } from "wouter-preact";
import type { Song } from "@/lib/music/model";
import { rowSelect } from "@/lib/music/views/hooks/row-select";
import { useWindow } from "@/lib/music/views/hooks/use-window";
import {
	currentSong,
	isLoading,
	isPlaying,
} from "@/lib/music/views/stores/audio";
import { playFromQueueSelection } from "@/lib/music/views/stores/player";
import {
	clearSelection,
	selectedIds,
	selectionActive,
	toggleSelection,
} from "@/lib/music/views/stores/selection";
import TrackActions, {
	buildActions,
} from "@/lib/music/views/ui/track/track-actions";
import TrackArt from "@/lib/music/views/ui/track/track-art";
import { t } from "@/lib/shared/i18n";
import { cover } from "@/lib/shared/utils/cover";
import { secondsToTime } from "@/lib/shared/utils/format";
import { cn } from "@/lib/shared/utils/tw";
import { usePointerDrag } from "@/lib/shared/views/hooks/use-pointer-drag";
import AlbumLink from "@/lib/shared/views/ui/components/album-link";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import { openContextMenu } from "@/lib/shared/views/ui/components/context-menu";
import { CoverFallback } from "@/lib/shared/views/ui/components/cover-fallback";
import { Image } from "@/lib/shared/views/ui/components/image";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

type Props = {
	songs: Song[];
	// sourceId identifies the playlist/album these songs belong to, so playback
	// can be tied back to its card.
	sourceId?: string;
	// resetKey changes when the listing itself changes (a filter or an order), so
	// the list goes back to the top instead of keeping an offset that now points
	// at a different song.
	resetKey?: string;
	onRemove?: (songs: Song[]) => void;
	onReorder?: (from: number, to: number) => void;
};

const ROW_PX = 64;

// Scroll offset per list, kept for the session: a list that is unmounted and
// mounted again (a route change, a cleared query) continues where it was.
const offsets = new Map<string, number>();

const dragFrom = signal<number | null>(null);
const dragOver = signal<number | null>(null);

const currentId = computed(() => currentSong.value?.id);

function endDrag() {
	dragFrom.value = null;
	dragOver.value = null;
}

function PlaylistRow({
	song,
	songs,
	index,
	sourceId,
	selected,
	reorderable,
	isDragSource,
	isDropTarget,
	removable,
	onRemove,
}: {
	song: Song;
	songs: Song[];
	index: number;
	sourceId?: string;
	selected: boolean;
	reorderable: boolean;
	isDragSource: boolean;
	isDropTarget: boolean;
	removable: boolean;
	onRemove?: (songs: Song[]) => void;
}) {
	const isCurrent = song.id === currentId.value;
	const broken = song.broken === true;
	const [, navigate] = useLocation();

	const { onClick } = rowSelect({
		song,
		songs,
		index,
		disabled: broken,
		onActivate: () => void playFromQueueSelection(songs, song, sourceId),
	});

	return (
		<div
			data-id={song.id}
			data-playlist-index={index}
			role="none"
			title={broken ? t("music.track.broken") : undefined}
			onClick={onClick}
			onContextMenu={(e) => {
				if (broken) return;
				const { actions } = buildActions({
					songs: [song],
					selection: selected,
					removable,
					onPlay: () => void playFromQueueSelection(songs, song, sourceId),
					onRemove,
					onToggleSelect: toggleSelection,
					navigate,
				});
				openContextMenu(actions, e);
			}}
			class={cn(
				"group absolute left-0 flex h-full w-full select-none items-center justify-between gap-2 border-b border-zinc-900 px-3 transition",
				broken
					? "cursor-not-allowed text-zinc-500"
					: "hover:cursor-pointer hover:bg-zinc-800/60",
				isCurrent ? "font-medium text-(--dominant-color)!" : "",
				selected && "bg-zinc-800/60",
				isDragSource && "opacity-40",
				isDropTarget &&
					"bg-(--dominant-color)/15 ring-1 ring-(--dominant-color)/50 ring-inset",
			)}
		>
			<div class="grid grid-cols-[auto_1fr] gap-2">
				<div class="relative shrink-0">
					<TrackArt
						song={song}
						current={isCurrent}
						loading={isLoading.value}
						playing={isPlaying.value}
						alt={song.name}
						class={
							isCurrent
								? "rounded-md outline-2 outline-(--dominant-color)"
								: undefined
						}
						imgClass="h-10 w-10 rounded-md"
					/>
					{reorderable && (
						<span
							class="pointer-events-none absolute top-1 left-1 flex size-5 items-center justify-center rounded-full border border-zinc-700 bg-zinc-950/80 text-zinc-300 opacity-0 transition-opacity group-hover:opacity-100"
							aria-hidden
						>
							<GripVertical size={13} />
						</span>
					)}
				</div>

				<div class="flex min-w-0 flex-col text-start">
					<span
						class={cn(
							"truncate text-sm",
							broken ? "text-zinc-500" : "text-white",
						)}
					>
						{song.name}
					</span>

					<span class="truncate text-xs text-zinc-400">
						{broken ? (
							<span class="inline-flex items-center gap-1 text-red-400/80">
								<TriangleAlert size={12} />
								{t("music.track.brokenLabel")}
							</span>
						) : (
							<>
								<ArtistLinks artists={song.artists} />
								{song.album?.title && song.album.id !== sourceId && (
									<>
										{" • "}
										<AlbumLink album={song.album} />
									</>
								)}
							</>
						)}
					</span>
				</div>
			</div>

			<div class="flex shrink-0 items-center gap-2">
				{!broken && (
					<TrackActions
						songs={[song]}
						selection={selected}
						removable={removable}
						onRemove={onRemove}
						onToggleSelect={toggleSelection}
						onPlay={() => void playFromQueueSelection(songs, song, sourceId)}
					/>
				)}
				<div className="tabular-nums text-xs text-zinc-500">
					{secondsToTime(song.duration)}
				</div>
			</div>
		</div>
	);
}

const PlaylistRowMemo = memo(PlaylistRow);

export function Virtualization({
	songs,
	sourceId,
	resetKey,
	onRemove,
	onReorder,
}: Props) {
	const { ref, totalSize, items } = useWindow<HTMLDivElement>(
		songs.length,
		ROW_PX,
	);
	const reorderable = !!onReorder && songs.length > 1;
	const selection = selectionActive.value;
	const selected = selectedIds.value;
	const dragOrigin = useRef<number | null>(null);

	const indexFromClientY = (clientY: number) => {
		const el = ref.current;
		if (!el || songs.length === 0) return 0;
		const rect = el.getBoundingClientRect();
		const rel = el.scrollTop + (clientY - rect.top);
		return Math.max(0, Math.min(songs.length - 1, Math.floor(rel / ROW_PX)));
	};

	const hover = (clientY: number) => {
		const next = indexFromClientY(clientY);
		if (dragOver.value !== next) dragOver.value = next;
	};

	const { start, captureClick } = usePointerDrag({
		scroll: ref,
		begin: () => {
			if (dragOrigin.current == null) return;
			dragFrom.value = dragOrigin.current;
			dragOver.value = dragOrigin.current;
		},
		move: (e) => hover(e.clientY),
		end: (dragged) => {
			const from = dragFrom.value;
			const to = dragOver.value;
			endDrag();
			if (dragged && from != null && to != null && from !== to) {
				onReorder?.(from, to);
			}
		},
	});

	const handlePointerDown = (e: PointerEvent) => {
		if (!reorderable || selection) return;
		const target = e.target as Element;
		if (target.closest("button")) return;
		const row = target.closest("[data-playlist-index]");
		if (!row) return;
		const index = Number(row.getAttribute("data-playlist-index"));
		if (!Number.isInteger(index)) return;
		if (!start(e)) return;
		dragOrigin.current = index;
	};

	// The list opens where it was left: a remount (a route change, a cleared
	// query) or a different playlist in place restores the saved offset instead
	// of stealing the scroll back to the playing song, and only a different
	// listing starts at the top.
	const shown = useRef<{ key?: string; source?: string }>({
		key: resetKey,
		source: sourceId,
	});
	useLayoutEffect(() => {
		const el = ref.current;
		if (!el) return;
		const previous = shown.current;
		shown.current = { key: resetKey, source: sourceId };
		if (previous.key !== resetKey) {
			if (sourceId) offsets.delete(sourceId);
			el.scrollTop = 0;
			return;
		}
		const saved = sourceId ? offsets.get(sourceId) : undefined;
		if (saved !== undefined) {
			el.scrollTop = saved;
			return;
		}
		// A different list starts at the top unless the playing song belongs here:
		// keeping the previous offset would leave the new list scrolled to a row that
		// means nothing. A height of 0 would compute an offset far below the real
		// position, so centering waits for a real measurement.
		const index = songs.findIndex((song) => song.id === currentId.value);
		if (index < 0) {
			if (previous.source !== sourceId) el.scrollTop = 0;
			return;
		}
		if (el.clientHeight === 0) return;
		el.scrollTop = index * ROW_PX - el.clientHeight / 2 + ROW_PX / 2;
	}, [resetKey, sourceId]);

	// Only a real list (a playlist, an album) is worth returning to: a search
	// result or an artist page is looked at once and left behind.
	useEffect(() => {
		const el = ref.current;
		if (!el || !sourceId) return;
		const save = () => offsets.set(sourceId, el.scrollTop);
		el.addEventListener("scroll", save, { passive: true });
		return () => el.removeEventListener("scroll", save);
	}, [sourceId]);

	const song = currentSong.value;
	const jumpTo = song ? songs.findIndex((s) => s.id === song.id) : -1;
	const first = items.length > 0 ? items[0].index : -1;
	const last = items.length > 0 ? items[items.length - 1].index : -1;
	const above = jumpTo >= 0 && jumpTo < first;
	const below = jumpTo >= 0 && jumpTo > last;

	const jump = () => {
		const el = ref.current;
		if (!el || jumpTo < 0) return;
		el.scrollTo({
			top: jumpTo * ROW_PX - el.clientHeight / 2 + ROW_PX / 2,
			behavior: "smooth",
		});
	};

	return (
		<div className="relative min-h-0 flex-1">
			{/* biome-ignore lint/a11y/noStaticElementInteractions: drag-to-reorder container */}
			{/* biome-ignore lint/a11y/useKeyWithClickEvents: empty area clears the selection */}
			<div
				ref={ref}
				className="h-full overflow-y-auto pr-3.75 rounded-lg border border-zinc-800 bg-zinc-950"
				onPointerDown={handlePointerDown}
				onClickCapture={captureClick}
				onClick={(e) => {
					const target = e.target as Element;
					if (!target.closest("[data-playlist-index]")) clearSelection();
				}}
			>
				{songs.length === 0 ? (
					<div className="flex min-h-32 flex-1 items-center justify-center px-4 py-8 text-sm text-zinc-500">
						{t("music.track.noSongs")}
					</div>
				) : (
					<div
						className="relative w-full pb-3"
						style={{
							height: `${totalSize}px`,
						}}
					>
						{items.map((virtualRow) => {
							const song = songs[virtualRow.index];

							return (
								<div
									key={`${song.id}-${virtualRow.index}`}
									data-playlist-index={virtualRow.index}
									class="absolute left-0 w-full"
									style={{
										height: `${virtualRow.size}px`,
										transform: `translateY(${virtualRow.start}px)`,
									}}
								>
									<PlaylistRowMemo
										song={song}
										songs={songs}
										index={virtualRow.index}
										sourceId={sourceId}
										selected={selected.has(song.id)}
										reorderable={reorderable}
										isDragSource={dragFrom.value === virtualRow.index}
										isDropTarget={
											dragFrom.value != null &&
											dragOver.value === virtualRow.index &&
											dragFrom.value !== virtualRow.index
										}
										removable={!!onRemove}
										onRemove={onRemove}
									/>
								</div>
							);
						})}
					</div>
				)}
			</div>

			<Scrollbar target={ref} />

			<button
				type="button"
				title={t("music.track.jumpToCurrent")}
				aria-label={t("music.track.jumpToCurrent")}
				onClick={jump}
				class={cn(
					"absolute bottom-3 left-1/2 z-10 flex -translate-x-1/2 cursor-pointer items-center gap-2 rounded-full border border-zinc-700 bg-zinc-900/90 py-1.5 pr-2.5 pl-1.5 text-xs text-zinc-300 shadow-lg transition-opacity hover:bg-zinc-800",
					above || below ? "opacity-100" : "pointer-events-none opacity-0",
				)}
			>
				<Image
					loading="lazy"
					src={cover(song?.album?.covers?.[0], 40)}
					alt=""
					class="size-5 rounded-full"
					fallback={<CoverFallback class="size-5 shrink-0 rounded-full" />}
				/>
				<span class="max-w-40 truncate">{song?.name}</span>
				{above ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
			</button>
		</div>
	);
}
