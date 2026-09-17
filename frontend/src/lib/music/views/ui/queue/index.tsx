import { signal } from "@preact/signals";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
	ArrowUpFromLine,
	ChevronDown,
	CircleAlert,
	GripVertical,
	ListMusic,
	Loader,
	Music,
	Pause,
	Play,
	Trash2,
} from "lucide-preact";
import { memo } from "preact/compat";
import { useEffect, useRef } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { useQueuePanel } from "@/lib/music/views/hooks/use-queue";
import {
	failedSongs,
	isLoading,
	isPlaying,
} from "@/lib/music/views/stores/audio";
import {
	moveAfterCurrent,
	moveQueue,
	playAt,
	unqueue,
} from "@/lib/music/views/stores/player";
import { queue, showQueue } from "@/lib/music/views/stores/queue";
import { cn } from "@/lib/shared/utils/tw";
import AlbumLink from "@/lib/shared/views/ui/components/album-link";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import { Modal } from "@/lib/shared/views/ui/components/modal";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

const ROW_PX = 64;
const QUEUE_VIEW_LOOKBACK = 1;

const dragFrom = signal<number | null>(null);
const dragOver = signal<number | null>(null);

function startDrag(e: DragEvent, globalIndex: number) {
	const target = e.target;
	if (target instanceof Element && target.closest("button") != null) {
		e.preventDefault();
		return;
	}
	if (queue.value.length <= 1) {
		e.preventDefault();
		return;
	}
	dragFrom.value = globalIndex;
	dragOver.value = globalIndex;
	const dt = e.dataTransfer;
	if (!dt) return;
	try {
		dt.setData("text/plain", String(globalIndex));
		dt.effectAllowed = "move";
	} catch {
		/* noop */
	}
}

function endDrag() {
	dragFrom.value = null;
	dragOver.value = null;
}

interface QueueRowProps {
	song: Song;
	globalIndex: number;
	songsLength: number;
	isCurrent: boolean;
	isPlaying: boolean;
	isLoading: boolean;
	failed: boolean;
	canSnapBelow: boolean;
	isDragSource: boolean;
	isDropTarget: boolean;
}

const QueueRow = memo(function QueueRow({
	song,
	globalIndex,
	songsLength,
	isCurrent,
	isPlaying: playing,
	isLoading: loading,
	failed,
	canSnapBelow,
	isDragSource,
	isDropTarget,
}: QueueRowProps) {
	const draggable = songsLength > 1;
	const canMoveDown = globalIndex < songsLength - 1;

	return (
		<div
			draggable={draggable}
			title={draggable ? "Drag row to reorder" : undefined}
			role="none"
			class={cn(
				"flex h-full w-full select-none items-center gap-1 border-b border-zinc-900/80 px-1 sm:gap-2 sm:px-2",
				isCurrent ? "bg-zinc-900/50" : "hover:bg-zinc-900/30",
				isDragSource && "opacity-40",
				isDropTarget &&
					"bg-(--dominant-color)/15 ring-1 ring-(--dominant-color)/40 ring-inset",
				draggable ? "cursor-grab active:cursor-grabbing" : "cursor-default",
			)}
			onDragStart={(e) => startDrag(e, globalIndex)}
			onDragEnd={endDrag}
		>
			<div
				class={cn(
					"hidden shrink-0 touch-none rounded p-0.5 text-zinc-600 sm:flex",
					songsLength <= 1 && "opacity-30",
				)}
				aria-hidden
			>
				<GripVertical size={16} />
			</div>
			<span class="w-5 shrink-0 text-center text-[11px] tabular-nums text-zinc-600 sm:w-6 sm:text-xs mr-3">
				{globalIndex + 1}
			</span>
			<div class="relative">
				<img
					loading="lazy"
					decoding="async"
					src={song.album?.covers?.[0]}
					alt=""
					class={cn(
						"h-9 w-9 shrink-0 rounded-md sm:h-10 sm:w-10",
						isCurrent && "ring-2 ring-(--dominant-color)/80",
						isCurrent && "brightness-40",
					)}
				/>

				{isCurrent &&
					(loading ? (
						<Loader
							size={23}
							class="absolute inset-0 z-10 m-auto animate-spin text-zinc-400 drop-shadow-md drop-shadow-black"
						/>
					) : playing ? (
						<Music
							size={25}
							class="absolute inset-0 z-10 m-auto text-(--dominant-color) drop-shadow-md drop-shadow-black"
						/>
					) : (
						<Pause
							size={25}
							class="absolute inset-0 z-10 m-auto fill-(--dominant-color) drop-shadow-md drop-shadow-black"
						/>
					))}
			</div>
			<div class="min-w-0 flex-1 pr-1">
				<div class="flex items-center gap-1.5">
					<p class="truncate text-[13px] leading-tight sm:text-sm text-zinc-100">
						{song.name}
					</p>
					{failed && (
						<span
							title="Could not play this track"
							class="shrink-0 text-red-400"
						>
							<CircleAlert size={14} />
						</span>
					)}
				</div>
				<p class="truncate text-[11px] text-zinc-500 sm:text-xs">
					<ArtistLinks artists={song.artists} />
					{song.album.title && (
						<>
							{" • "}
							<AlbumLink album={song.album} />
						</>
					)}
				</p>
			</div>

			<div class="flex shrink-0 items-center gap-0 sm:gap-0.5">
				<button
					type="button"
					draggable={false}
					title="Play now"
					disabled={isCurrent}
					class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-(--dominant-color) disabled:cursor-not-allowed disabled:opacity-30"
					onClick={() => void playAt(globalIndex)}
				>
					<Play size={17} />
				</button>
				<button
					type="button"
					draggable={false}
					title="Move down one position"
					disabled={!canMoveDown}
					class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-white disabled:cursor-not-allowed disabled:opacity-30"
					onClick={() => void moveQueue(globalIndex, globalIndex + 1)}
				>
					<ChevronDown size={17} />
				</button>
				<button
					type="button"
					draggable={false}
					title="Place just below the playing track"
					disabled={!canSnapBelow}
					class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-white disabled:cursor-not-allowed disabled:opacity-30"
					onClick={() => void moveAfterCurrent(globalIndex)}
				>
					<ArrowUpFromLine size={17} />
				</button>
				<button
					type="button"
					draggable={false}
					title="Remove from queue"
					class="cursor-pointer rounded-md p-1.5 text-zinc-500 transition hover:bg-red-950/50 hover:text-red-300"
					onClick={() => void unqueue(globalIndex)}
				>
					<Trash2 size={17} />
				</button>
			</div>
		</div>
	);
});

function Queue() {
	const panel = useQueuePanel();
	const parentRef = useRef<HTMLUListElement>(null);

	const songs = panel.songs;
	const idx = panel.currentIndex;
	const viewStart = idx >= 0 ? Math.max(0, idx - QUEUE_VIEW_LOOKBACK) : 0;
	const viewCount = songs.length === 0 ? 0 : songs.length - viewStart;

	const closeQueue = () => {
		showQueue.value = false;
	};

	const indexFromClientY = (clientY: number) => {
		const el = parentRef.current;
		if (!el || viewCount === 0) return viewStart;
		const r = el.getBoundingClientRect();
		const rel = el.scrollTop + (clientY - r.top);
		const local = Math.max(
			0,
			Math.min(viewCount - 1, Math.floor(rel / ROW_PX)),
		);
		return viewStart + local;
	};

	const handleDragOverCapture = (e: DragEvent) => {
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
		const next = indexFromClientY(e.clientY);
		if (dragOver.value !== next) {
			dragOver.value = next;
		}
	};

	const handleDrop = (e: DragEvent) => {
		if (dragFrom.value == null) return;
		e.preventDefault();
		const from = dragFrom.value;
		const to = indexFromClientY(e.clientY);
		endDrag();
		if (from !== to) {
			void moveQueue(from, to);
		}
	};

	const rowVirtualizer = useVirtualizer({
		count: viewCount,
		getScrollElement: () => parentRef.current,
		estimateSize: () => ROW_PX,
		overscan: 5,
		getItemKey: (localIndex) => {
			const g = viewStart + localIndex;
			return `${songs[g]?.id ?? "x"}-${g}`;
		},
	});

	useEffect(() => {
		if (!panel.open) return;
		const onKey = (e: KeyboardEvent) => {
			if (e.key === "Escape") {
				showQueue.value = false;
			}
		};
		window.addEventListener("keydown", onKey);
		return () => window.removeEventListener("keydown", onKey);
	}, [panel.open]);

	return (
		<Modal
			open={panel.open}
			close={closeQueue}
			labelledBy="queue-panel-title"
			closeLabel="Close queue"
		>
			<header class="flex shrink-0 flex-col gap-1 border-b border-zinc-800 px-4 py-3">
				<div class="flex items-center justify-between gap-2 text-white">
					<div class="flex items-center gap-2">
						<ListMusic size={22} class="text-(--dominant-color)" />
						<h2 id="queue-panel-title" class="sr-only">
							Cola
						</h2>
						<span class="text-xs text-zinc-500 tabular-nums">
							{songs.length === 0
								? "empty"
								: `${songs.length} tema${songs.length === 1 ? "" : "s"}`}
						</span>
					</div>
				</div>
			</header>

			<div class="relative flex min-h-0 flex-1 flex-col">
				<ul
					ref={parentRef}
					class={cn(
						"min-h-0 flex-1 overflow-y-auto px-2 pb-3 pt-1",
						dragFrom.value != null && "[&_button]:pointer-events-none",
					)}
					aria-label="Queue"
					onDragOverCapture={handleDragOverCapture}
					onDrop={handleDrop}
					onDragEnd={endDrag}
				>
					{songs.length === 0 ? (
						<li class="flex min-h-40 list-none flex-col items-center justify-center gap-2 px-4 py-10 text-center text-sm text-zinc-500">
							<p>Empty queue.</p>
							<p class="text-xs text-zinc-600">
								Pick a playlist and click a track to start.
							</p>
						</li>
					) : (
						<div
							class="relative w-full"
							style={{
								height: `${rowVirtualizer.getTotalSize()}px`,
							}}
						>
							{rowVirtualizer.getVirtualItems().map((virtualRow) => {
								const globalIndex = viewStart + virtualRow.index;
								const song = songs[globalIndex];
								const isCurrent = globalIndex === idx;

								return (
									<div
										key={virtualRow.key}
										class="absolute left-0 w-full"
										style={{
											height: `${virtualRow.size}px`,
											transform: `translateY(${virtualRow.start}px)`,
										}}
									>
										<QueueRow
											song={song}
											globalIndex={globalIndex}
											songsLength={songs.length}
											isCurrent={isCurrent}
											isPlaying={isCurrent && isPlaying.value}
											isLoading={isCurrent && isLoading.value}
											failed={failedSongs.value.has(song.id)}
											canSnapBelow={
												idx >= 0 &&
												idx < songs.length &&
												!isCurrent &&
												globalIndex !== idx + 1
											}
											isDragSource={dragFrom.value === globalIndex}
											isDropTarget={
												dragFrom.value != null &&
												dragOver.value === globalIndex &&
												dragFrom.value !== globalIndex
											}
										/>
									</div>
								);
							})}
						</div>
					)}
				</ul>
				<Scrollbar target={parentRef} />
			</div>
		</Modal>
	);
}

export default Queue;
