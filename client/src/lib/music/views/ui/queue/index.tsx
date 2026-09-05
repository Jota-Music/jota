import { useSignal } from "@preact/signals";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
	ArrowUpFromLine,
	ChevronDown,
	GripVertical,
	ListMusic,
	Loader,
	Music,
	Pause,
	Play,
	Trash2,
	X,
} from "lucide-preact";
import { useEffect, useRef } from "preact/hooks";
import { useQueuePanel } from "@/lib/music/views/hooks/use-queue";
import { isLoading, isPlaying } from "@/lib/music/views/stores/audio";
import { showQueue } from "@/lib/music/views/stores/queue";
import { cn } from "@/lib/shared/utils/tw";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";

const ROW_PX = 64;
const QUEUE_VIEW_LOOKBACK = 1;

function Queue() {
	const panel = useQueuePanel();
	const parentRef = useRef<HTMLUListElement>(null);
	const dragFrom = useSignal<number | null>(null);
	const dragOver = useSignal<number | null>(null);

	const songs = panel.songs;
	const idx = panel.currentIndex;
	const viewStart = idx >= 0 ? Math.max(0, idx - QUEUE_VIEW_LOOKBACK) : 0;
	const viewCount = songs.length === 0 ? 0 : songs.length - viewStart;

	const handleMaskClick = () => {
		showQueue.value = false;
	};

	const handleCloseClick = () => {
		showQueue.value = false;
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
		clearDrag();
		if (from !== to) {
			void panel.moveQueueItem(from, to);
		}
	};

	const handleDragEnd = () => {
		clearDrag();
	};

	const handleDragStart = (e: DragEvent, globalIndex: number) => {
		const target = e.target;
		if (target instanceof Element && target.closest("button") != null) {
			e.preventDefault();
			return;
		}
		if (songs.length <= 1) {
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
	};

	const handlePlayClick = (globalIndex: number) => {
		void panel.playSongAtQueueIndex(globalIndex);
	};

	const handleMoveBelowClick = (globalIndex: number) => {
		void panel.moveJustBelowCurrentPlaying(globalIndex);
	};

	const handleMoveDownClick = (globalIndex: number) => {
		void panel.moveQueueItem(globalIndex, globalIndex + 1);
	};

	const handleRemoveFromQueueClick = (globalIndex: number) => {
		void panel.removeFromQueueAt(globalIndex);
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

	const clearDrag = () => {
		dragFrom.value = null;
		dragOver.value = null;
	};

	const rowVirtualizer = useVirtualizer({
		count: viewCount,
		getScrollElement: () => parentRef.current,
		estimateSize: () => ROW_PX,
		overscan: 10,
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

	if (!panel.open) {
		return null;
	}

	return (
		<div
			class="fixed inset-0 z-50 flex items-end justify-center sm:items-center p-0 sm:p-4"
			role="presentation"
		>
			<button
				type="button"
				class="absolute inset-0 bg-black/70 backdrop-blur-[2px]"
				aria-label="Cerrar cola"
				onClick={handleMaskClick}
			/>

			<div
				class="relative z-10 flex h-[80dvh] w-full max-w-2xl flex-col overflow-hidden rounded-t-2xl border border-zinc-800 bg-zinc-950 shadow-2xl sm:rounded-2xl"
				role="dialog"
				aria-modal="true"
				aria-labelledby="queue-panel-title"
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
									? "vacía"
									: `${songs.length} tema${songs.length === 1 ? "" : "s"}`}
							</span>
						</div>
						<button
							type="button"
							class="rounded-lg p-2 text-zinc-400 transition hover:bg-zinc-800 hover:text-white cursor-pointer"
							aria-label="Cerrar"
							onClick={handleCloseClick}
						>
							<X size={20} />
						</button>
					</div>
				</header>

				<ul
					ref={parentRef}
					class={cn(
						"min-h-0 flex-1 overflow-y-auto px-2 pb-3 pt-1",
						dragFrom.value != null && "[&_button]:pointer-events-none",
					)}
					aria-label="Queue"
					onDragOverCapture={handleDragOverCapture}
					onDrop={handleDrop}
					onDragEnd={handleDragEnd}
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
								const localIndex = virtualRow.index;
								const globalIndex = viewStart + localIndex;
								const song = songs[globalIndex];
								const isCurrent = globalIndex === idx;
								const canSnapBelow =
									idx >= 0 &&
									idx < songs.length &&
									!isCurrent &&
									globalIndex !== idx + 1;
								const canMoveDown = globalIndex < songs.length - 1;
								const isDragSource = dragFrom.value === globalIndex;
								const isDropTarget =
									dragFrom.value != null &&
									dragOver.value === globalIndex &&
									dragFrom.value !== globalIndex;

								return (
									<div
										key={virtualRow.key}
										draggable={songs.length > 1}
										title={songs.length > 1 ? "Drag row to reorder" : undefined}
										role="none"
										class={cn(
											"absolute left-0 flex w-full select-none items-center gap-1 border-b border-zinc-900/80 px-1 sm:gap-2 sm:px-2",
											isCurrent ? "bg-zinc-900/50" : "hover:bg-zinc-900/30",
											isDragSource && "opacity-40",
											isDropTarget &&
												"bg-(--dominant-color)/15 ring-1 ring-(--dominant-color)/40 ring-inset",
											songs.length > 1 && "cursor-grab active:cursor-grabbing",
											songs.length <= 1 && "cursor-default",
										)}
										style={{
											height: `${virtualRow.size}px`,
											transform: `translateY(${virtualRow.start}px)`,
										}}
										onDragStart={(e) => handleDragStart(e, globalIndex)}
										onDragEnd={handleDragEnd}
									>
										<div
											class={cn(
												"flex shrink-0 touch-none rounded p-0.5 text-zinc-600",
												songs.length <= 1 && "opacity-30",
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
												(isLoading.value ? (
													<Loader
														size={23}
														class="absolute inset-0 z-10 m-auto animate-spin text-zinc-400 drop-shadow-md drop-shadow-black"
													/>
												) : isPlaying.value ? (
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
											<p class="truncate text-[13px] leading-tight sm:text-sm text-zinc-100">
												{song.name}
											</p>
											<p class="truncate text-[11px] text-zinc-500 sm:text-xs">
												<ArtistLinks artists={song.artists} />
											</p>
										</div>

										<div class="flex shrink-0 items-center gap-0 sm:gap-0.5">
											<button
												type="button"
												draggable={false}
												title="Reproducir ahora"
												disabled={isCurrent}
												class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-(--dominant-color) disabled:cursor-not-allowed disabled:opacity-30"
												onClick={() => handlePlayClick(globalIndex)}
											>
												<Play size={17} />
											</button>
											<button
												type="button"
												draggable={false}
												title="Bajar una posición"
												disabled={!canMoveDown}
												class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-white disabled:cursor-not-allowed disabled:opacity-30"
												onClick={() => handleMoveDownClick(globalIndex)}
											>
												<ChevronDown size={17} />
											</button>
											<button
												type="button"
												draggable={false}
												title="Poner justo debajo del tema en reproducción"
												disabled={!canSnapBelow}
												class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-white disabled:cursor-not-allowed disabled:opacity-30"
												onClick={() => handleMoveBelowClick(globalIndex)}
											>
												<ArrowUpFromLine size={17} />
											</button>
											<button
												type="button"
												draggable={false}
												title="Remove from queue"
												class="cursor-pointer rounded-md p-1.5 text-zinc-500 transition hover:bg-red-950/50 hover:text-red-300"
												onClick={() => handleRemoveFromQueueClick(globalIndex)}
											>
												<Trash2 size={17} />
											</button>
										</div>
									</div>
								);
							})}
						</div>
					)}
				</ul>
			</div>
		</div>
	);
}

export default Queue;
