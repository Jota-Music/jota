import { computed, signal } from "@preact/signals";
import { GripVertical, TriangleAlert } from "lucide-preact";
import { memo } from "preact/compat";
import { useRef } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { coarse, rowSelect } from "@/lib/music/views/hooks/row-select";
import { useWindow } from "@/lib/music/views/hooks/use-window";
import {
	currentSong,
	isLoading,
	isPlaying,
} from "@/lib/music/views/stores/audio";
import { playFromQueueSelection } from "@/lib/music/views/stores/player";
import {
	clearSelection,
	selectedSongs,
	selectionActive,
	toggleSelection,
} from "@/lib/music/views/stores/selection";
import TrackActions, {
	buildActions,
} from "@/lib/music/views/ui/track/track-actions";
import TrackArt from "@/lib/music/views/ui/track/track-art";
import { t } from "@/lib/shared/i18n";
import { secondsToTime } from "@/lib/shared/utils/format";
import { cn } from "@/lib/shared/utils/tw";
import { usePointerDrag } from "@/lib/shared/views/hooks/use-pointer-drag";
import AlbumLink from "@/lib/shared/views/ui/components/album-link";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import { openContextMenu } from "@/lib/shared/views/ui/components/context-menu";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

type Props = {
	songs: Song[];
	onRemove?: (songs: Song[]) => void;
	onReorder?: (from: number, to: number) => void;
};

const ROW_PX = 64;

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
	selected: boolean;
	reorderable: boolean;
	isDragSource: boolean;
	isDropTarget: boolean;
	removable: boolean;
	onRemove?: (songs: Song[]) => void;
}) {
	const isCurrent = song.id === currentId.value;
	const broken = song.broken === true;

	const { onClick } = rowSelect({
		song,
		songs,
		index,
		disabled: broken,
		onActivate: () => void playFromQueueSelection(songs, song),
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
					onRemove,
					onToggleSelect: toggleSelection,
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
							class={cn(
								"pointer-events-none absolute top-1 left-1 flex size-5 items-center justify-center rounded-full border border-zinc-700 bg-zinc-950/80 text-zinc-300",
								coarse
									? "opacity-100"
									: "opacity-0 transition-opacity group-hover:opacity-100",
							)}
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
								{song.album?.title && (
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

export function Virtualization({ songs, onRemove, onReorder }: Props) {
	const { ref, totalSize, items } = useWindow<HTMLDivElement>(
		songs.length,
		ROW_PX,
	);
	const reorderable = !!onReorder && songs.length > 1;
	const selection = selectionActive.value;
	const selected = new Set(selectedSongs.value.map((s) => s.id));
	const source = useRef<number | null>(null);

	const indexFromClientY = (clientY: number) => {
		const el = ref.current;
		if (!el || songs.length === 0) return 0;
		const rect = el.getBoundingClientRect();
		const rel = el.scrollTop + (clientY - rect.top);
		return Math.max(0, Math.min(songs.length - 1, Math.floor(rel / ROW_PX)));
	};

	const { start, captureClick } = usePointerDrag({
		scroll: ref,
		begin: () => {
			if (source.current == null) return;
			dragFrom.value = source.current;
			dragOver.value = source.current;
		},
		move: (e) => {
			const next = indexFromClientY(e.clientY);
			if (dragOver.value !== next) dragOver.value = next;
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

	const handlePointerDown = (e: PointerEvent) => {
		if (!reorderable || selection) return;
		const target = e.target as Element;
		if (target.closest("button")) return;
		const row = target.closest("[data-playlist-index]");
		if (!row) return;
		const index = Number(row.getAttribute("data-playlist-index"));
		if (!Number.isInteger(index)) return;
		if (!start(e)) return;
		source.current = index;
	};

	const handleDragOverCapture = (e: DragEvent) => {
		if (dragFrom.value == null) return;
		e.preventDefault();
		const next = indexFromClientY(e.clientY);
		if (dragOver.value !== next) dragOver.value = next;
	};

	const handleDrop = (e: DragEvent) => {
		if (dragFrom.value == null) return;
		e.preventDefault();
		const from = dragFrom.value;
		const to = indexFromClientY(e.clientY);
		endDrag();
		if (from !== to) onReorder?.(from, to);
	};

	const startDrag = (e: DragEvent, index: number) => {
		if (!reorderable || selection) {
			e.preventDefault();
			return;
		}
		dragFrom.value = index;
		const dt = e.dataTransfer;
		if (!dt) return;
		try {
			dt.setData("text/plain", String(index));
			dt.effectAllowed = "move";
		} catch {
			/* noop */
		}
	};

	return (
		<div className="relative min-h-0 flex-1">
			{/* biome-ignore lint/a11y/noStaticElementInteractions: drag-to-reorder container */}
			{/* biome-ignore lint/a11y/useKeyWithClickEvents: empty area clears the selection */}
			<div
				ref={ref}
				className="h-full overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950"
				onPointerDown={handlePointerDown}
				onClickCapture={captureClick}
				onClick={(e) => {
					const target = e.target as Element;
					if (!target.closest("[data-playlist-index]")) clearSelection();
				}}
				onDragOverCapture={handleDragOverCapture}
				onDrop={handleDrop}
				onDragEnd={endDrag}
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
								// biome-ignore lint/a11y/noStaticElementInteractions: draggable row wrapper
								<div
									key={`${song.id}-${virtualRow.index}`}
									data-playlist-index={virtualRow.index}
									class="absolute left-0 w-full"
									style={{
										height: `${virtualRow.size}px`,
										transform: `translateY(${virtualRow.start}px)`,
									}}
									draggable={reorderable && coarse}
									onDragStart={(e) => startDrag(e, virtualRow.index)}
								>
									<PlaylistRowMemo
										song={song}
										songs={songs}
										index={virtualRow.index}
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
		</div>
	);
}
