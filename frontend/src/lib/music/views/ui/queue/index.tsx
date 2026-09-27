import { signal } from "@preact/signals";
import {
	ArrowUpFromLine,
	ChevronDown,
	CircleAlert,
	GripVertical,
	ListMusic,
	MoreVertical,
	Trash2,
} from "lucide-preact";
import { memo } from "preact/compat";
import { useEffect, useRef } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { useWindow } from "@/lib/music/views/hooks/use-window";
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
import { currentIndex, queue, showQueue } from "@/lib/music/views/stores/queue";
import TrackArt from "@/lib/music/views/ui/track/track-art";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { usePointerDrag } from "@/lib/shared/views/hooks/use-pointer-drag";
import AlbumLink from "@/lib/shared/views/ui/components/album-link";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import {
	type Action,
	openContextMenu,
} from "@/lib/shared/views/ui/components/context-menu";
import { Modal, ModalHeader } from "@/lib/shared/views/ui/components/modal";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

const ROW_PX = 64;
const QUEUE_VIEW_LOOKBACK = 1;
const COARSE = window.matchMedia("(pointer: coarse)").matches;

const dragFrom = signal<number | null>(null);
const dragOver = signal<number | null>(null);

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
	const reorderable = songsLength > 1;
	const canMoveDown = globalIndex < songsLength - 1;

	const actions = (): Action[] => {
		const out: Action[] = [];
		if (canMoveDown) {
			out.push({
				icon: ChevronDown,
				label: t("music.queue.moveDown"),
				run: () => void moveQueue(globalIndex, globalIndex + 1),
			});
		}
		if (canSnapBelow) {
			out.push({
				icon: ArrowUpFromLine,
				label: t("music.queue.snapBelow"),
				run: () => void moveAfterCurrent(globalIndex),
			});
		}
		out.push({
			icon: Trash2,
			label: t("music.queue.remove"),
			danger: true,
			run: () => void unqueue(globalIndex),
		});
		return out;
	};

	return (
		<div
			data-drag-self={reorderable || undefined}
			title={reorderable ? t("music.queue.drag") : undefined}
			role="none"
			class={cn(
				"flex h-full w-full cursor-pointer select-none items-center gap-2 border-b border-zinc-900/80 px-1 sm:px-2",
				isCurrent ? "bg-zinc-900/50" : "hover:bg-zinc-900/30",
				// The queue captures the pointer on the list, not the row, so the
				// closed hand is set on the rows themselves.
				dragFrom.value != null && "cursor-grabbing!",
				isDragSource && "opacity-40",
				isDropTarget &&
					"bg-(--dominant-color)/15 ring-1 ring-(--dominant-color)/40 ring-inset",
			)}
			onClick={(e) => {
				if ((e.target as Element).closest("button, a")) return;
				void playAt(globalIndex);
			}}
			onContextMenu={(e) => {
				// Touch reaches the same actions through the row's menu button.
				if (COARSE) {
					e.preventDefault();
					return;
				}
				openContextMenu(actions(), e);
			}}
		>
			<span class="w-5 shrink-0 text-center text-[11px] tabular-nums text-zinc-600 sm:w-6 sm:text-xs mr-3">
				{globalIndex + 1}
			</span>
			<TrackArt
				song={song}
				current={isCurrent}
				loading={loading}
				playing={playing}
				imgClass={cn(
					"h-9 w-9 shrink-0 rounded-md sm:h-10 sm:w-10",
					isCurrent && "ring-2 ring-(--dominant-color)/80",
				)}
			/>
			<div class="min-w-0 flex-1 pr-1">
				<div class="flex items-center gap-1.5">
					<p class="truncate text-[13px] leading-tight sm:text-sm text-zinc-100">
						{song.name}
					</p>
					{failed && (
						<span title={t("music.queue.failed")} class="shrink-0 text-red-400">
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

			{/* Touch has no room for three buttons per row, so it gets the same
			    actions behind one menu button. */}
			<div class="hidden shrink-0 items-center gap-0.5 sm:flex">
				<button
					type="button"
					draggable={false}
					title={t("music.queue.moveDown")}
					disabled={!canMoveDown}
					class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-white disabled:cursor-not-allowed disabled:opacity-30"
					onClick={() => void moveQueue(globalIndex, globalIndex + 1)}
				>
					<ChevronDown size={17} />
				</button>
				<button
					type="button"
					draggable={false}
					title={t("music.queue.snapBelow")}
					disabled={!canSnapBelow}
					class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-white disabled:cursor-not-allowed disabled:opacity-30"
					onClick={() => void moveAfterCurrent(globalIndex)}
				>
					<ArrowUpFromLine size={17} />
				</button>
				<button
					type="button"
					draggable={false}
					title={t("music.queue.remove")}
					class="cursor-pointer rounded-md p-1.5 text-zinc-500 transition hover:bg-red-950/50 hover:text-red-300"
					onClick={() => void unqueue(globalIndex)}
				>
					<Trash2 size={17} />
				</button>
			</div>

			<button
				type="button"
				draggable={false}
				title={t("music.queue.actions")}
				aria-label={t("music.queue.actions")}
				class="flex shrink-0 cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-white sm:hidden"
				onClick={(e) => openContextMenu(actions(), e)}
			>
				<MoreVertical size={17} />
			</button>

			{/* Desktop drags from anywhere, so the grip is only decorative there.
			    Touch drags from the grip alone, leaving the artist and album
			    links tappable. */}
			<div
				data-drag-handle
				class={cn(
					"ml-0.5 flex shrink-0 touch-none rounded p-0.5 text-zinc-600",
					songsLength <= 1 && "opacity-30",
				)}
				aria-hidden
			>
				<GripVertical size={16} />
			</div>
		</div>
	);
});

function Queue() {
	const songs = queue.value;
	const idx = currentIndex.value;
	const viewStart =
		idx >= 0
			? Math.min(Math.max(0, idx - QUEUE_VIEW_LOOKBACK), songs.length)
			: 0;
	const viewCount = songs.length - viewStart;

	const { ref, totalSize, items } = useWindow<HTMLUListElement>(
		viewCount,
		ROW_PX,
	);

	const closeQueue = () => {
		showQueue.value = false;
	};

	const indexFromClientY = (clientY: number) => {
		const el = ref.current;
		if (!el || viewCount === 0) return viewStart;
		const r = el.getBoundingClientRect();
		const rel = el.scrollTop + (clientY - r.top);
		const local = Math.max(
			0,
			Math.min(viewCount - 1, Math.floor(rel / ROW_PX)),
		);
		return viewStart + local;
	};

	const hover = (clientY: number) => {
		const next = indexFromClientY(clientY);
		if (dragOver.value !== next) {
			dragOver.value = next;
		}
	};

	const source = useRef<number | null>(null);

	const { start, captureClick } = usePointerDrag({
		scroll: ref,
		touch: true,
		begin: () => {
			if (source.current == null) return;
			dragFrom.value = source.current;
			dragOver.value = source.current;
		},
		move: (e) => hover(e.clientY),
		end: (dragged) => {
			const from = dragFrom.value;
			const to = dragOver.value;
			endDrag();
			if (dragged && from != null && to != null && from !== to) {
				void moveQueue(from, to);
			}
		},
	});

	const handlePointerDown = (e: PointerEvent) => {
		if (songs.length <= 1) return;
		const target = e.target as Element;
		if (target.closest("button")) return;
		// A finger has no hover to aim with, so it only drags from the grip; the
		// rest of the row stays tappable down to the artist and album links.
		if (e.pointerType !== "mouse" && !target.closest("[data-drag-handle]"))
			return;
		const row = target.closest("[data-queue-index]");
		if (!row) return;
		const index = Number(row.getAttribute("data-queue-index"));
		if (!Number.isInteger(index)) return;
		if (!start(e)) return;
		source.current = index;
	};

	useEffect(() => {
		if (!showQueue.value) return;
		const onKey = (e: KeyboardEvent) => {
			if (e.key === "Escape") {
				showQueue.value = false;
			}
		};
		window.addEventListener("keydown", onKey);
		return () => window.removeEventListener("keydown", onKey);
	}, [showQueue.value]);

	return (
		<Modal
			open={showQueue.value}
			close={closeQueue}
			labelledBy="queue-panel-title"
		>
			<ModalHeader>
				<ListMusic size={22} class="shrink-0 text-(--dominant-color)" />
				<h2 id="queue-panel-title" class="sr-only">
					{t("music.queue.title")}
				</h2>
				<span class="text-xs text-zinc-500 tabular-nums">
					{songs.length === 0
						? t("music.queue.emptyShort")
						: t("music.trackCount", { count: songs.length })}
				</span>
			</ModalHeader>

			<div class="relative flex min-h-0 flex-1 flex-col">
				<ul
					ref={ref}
					class={cn(
						"min-h-0 flex-1 overflow-y-auto pl-2 pr-[15px] pb-3 pt-1",
						dragFrom.value != null && "[&_button]:pointer-events-none",
					)}
					aria-label={t("music.queue.aria")}
					onPointerDown={handlePointerDown}
					onClickCapture={captureClick}
				>
					{songs.length === 0 ? (
						<li class="flex min-h-40 list-none flex-col items-center justify-center gap-2 px-4 py-10 text-center text-sm text-zinc-500">
							<p>{t("music.queue.empty")}</p>
							<p class="text-xs text-zinc-600">{t("music.queue.emptyHint")}</p>
						</li>
					) : (
						<div
							class="relative w-full"
							style={{
								height: `${totalSize}px`,
							}}
						>
							{items.map((virtualRow) => {
								const globalIndex = viewStart + virtualRow.index;
								const song = songs[globalIndex];
								const isCurrent = globalIndex === idx;

								return (
									<div
										key={`${song.id}-${globalIndex}`}
										data-queue-index={globalIndex}
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
				<Scrollbar target={ref} />
			</div>
		</Modal>
	);
}

export default Queue;
