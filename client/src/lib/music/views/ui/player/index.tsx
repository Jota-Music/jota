import { signal } from "@preact/signals";
import {
	ChevronDown,
	ChevronUp,
	ListMusic,
	SkipBack,
	SkipForward,
} from "lucide-preact";
import type { ComponentChildren } from "preact";
import { useEffect, useRef } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { usePlayer } from "@/lib/music/views/hooks/use-player";
import { playerSkeletonOn, showQueue } from "@/lib/music/views/stores/queue";
import SaveYoutubeId from "@/lib/music/views/ui/player/save-youtube-id";
import { PlayerSkeleton } from "@/lib/music/views/ui/player/skeleton";
import Toggle from "@/lib/music/views/ui/player/toggle";
import VolumeControl from "@/lib/music/views/ui/volume";
import { secondsToTime } from "@/lib/shared/utils/format";
import AlbumLink from "@/lib/shared/views/ui/components/album-link";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import CircularProgress from "@/lib/shared/views/ui/components/circular-progress";
import Progress from "@/lib/shared/views/ui/components/progress";

const playerModalOpen = signal(false);

interface FullPlayerProps {
	song: Song;
	cover: string;
	isLoading: boolean;
	isPlaying: boolean;
	progress: number;
	duration: number;
	seek: (seconds: number) => void;
	toggleSong: () => Promise<void>;
	nextSong: () => Promise<void>;
	prevSong: () => Promise<void>;
	canPrev: boolean;
	canNext: boolean;
}

function FullPlayerContent({
	song,
	cover,
	isLoading,
	isPlaying,
	progress,
	duration,
	seek,
	toggleSong,
	nextSong,
	prevSong,
	canPrev,
	canNext,
}: FullPlayerProps) {
	return (
		<section class="w-full max-w-2xl mx-auto relative z-10 flex flex-col md:flex-row gap-4 md:gap-6 items-start justify-center text-(--dominant-color)">
			<div class="relative flex max-h-max w-full md:w-max">
				<CircularProgress
					value={progress}
					max={duration}
					min={0}
					onChange={seek}
					class="text-current mx-auto w-48 drop-shadow-lg drop-shadow-black"
				>
					<div class="relative aspect-square w-full">
						<img
							src={cover}
							alt={song.name}
							draggable={false}
							class="absolute inset-0 h-full w-full object-cover rounded-2xl shadow-lg opacity-40 animate-[spin_120s_linear_infinite] select-none"
							style="-webkit-user-drag: none"
						/>

						<div class="absolute inset-0 w-max h-max flex items-center justify-center gap-6 m-auto">
							<button
								type="button"
								disabled={!canPrev}
								onClick={() => void prevSong()}
								class="p-1 rounded-full transition drop-shadow-lg drop-shadow-black disabled:brightness-60 disabled:cursor-not-allowed cursor-pointer"
							>
								<SkipBack size={25} class="fill-current shadow-lg" />
							</button>

							<Toggle
								loading={isLoading}
								playing={isPlaying}
								onClick={() => void toggleSong()}
								size={25}
								class="bg-(--dominant-color)"
							/>

							<button
								type="button"
								disabled={!canNext}
								onClick={() => void nextSong()}
								class="p-1 rounded-full transition drop-shadow-lg drop-shadow-black disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
							>
								<SkipForward size={25} class="fill-current shadow-lg" />
							</button>
						</div>
					</div>
				</CircularProgress>
			</div>

			<div class="flex-1 min-w-0 w-full">
				<small class="text-white text-xs opacity-60 block truncate">
					<AlbumLink album={song.album} />
				</small>

				<h1 class="text-white text-xl font-semibold truncate">{song.name}</h1>

				<p class="text-white text-sm opacity-70 truncate">
					<ArtistLinks artists={song.artists} />
				</p>

				<SaveYoutubeId song={song} isPlaying={isPlaying} />

				<div class="mt-3 flex gap-2 justify-between items-center">
					<span class="text-white text-sm opacity-70 tabular-nums shrink-0">
						{secondsToTime(progress)}
					</span>

					<Progress
						value={progress}
						max={duration}
						min={0}
						onChange={seek}
						class="h-1.5 min-w-0 flex-1 opacity-80"
					/>

					<span class="text-white text-sm opacity-70 tabular-nums shrink-0">
						{secondsToTime(duration)}
					</span>
				</div>

				<div class="flex items-center justify-between gap-4 mt-4 pb-2">
					<Toggle
						loading={isLoading}
						playing={isPlaying}
						onClick={() => void toggleSong()}
						size={18}
						class="bg-(--dominant-color)"
					/>

					<VolumeControl />

					<button
						type="button"
						title="Cola de reproducción"
						aria-expanded={showQueue.value}
						onClick={() => {
							showQueue.value = !showQueue.value;
						}}
						class="rounded-md p-1 transition hover:bg-white/10 cursor-pointer"
					>
						<ListMusic size={25} class="fill-current" />
					</button>
				</div>
			</div>
		</section>
	);
}

function DraggableSheet({
	children,
	onClose,
}: {
	children: ComponentChildren;
	onClose: () => void;
}) {
	const sheetRef = useRef<HTMLDivElement>(null);
	const backdropRef = useRef<HTMLButtonElement>(null);
	const handleRef = useRef<HTMLButtonElement>(null);
	const contentRef = useRef<HTMLDivElement>(null);
	const dragStartY = useRef(0);
	const dragStartTime = useRef(0);
	const isDragging = useRef(false);
	const translateY = useRef(0);
	const onCloseRef = useRef(onClose);
	onCloseRef.current = onClose;

	const CLOSE_THRESHOLD = 120;
	const FLING_VELOCITY = 0.5;

	function closeSheet() {
		if (sheetRef.current) {
			sheetRef.current.style.transition =
				"transform 0.3s cubic-bezier(0.32, 0.72, 0, 1)";
			sheetRef.current.style.transform = "translateY(100%)";
		}
		if (backdropRef.current) {
			backdropRef.current.style.transition = "opacity 0.25s ease";
			backdropRef.current.style.opacity = "0";
		}
		setTimeout(() => {
			onCloseRef.current();
			translateY.current = 0;
		}, 280);
	}

	function snapBack() {
		if (sheetRef.current) {
			sheetRef.current.style.transition =
				"transform 0.4s cubic-bezier(0.32, 0.72, 0, 1)";
			sheetRef.current.style.transform = "translateY(0)";
		}
		if (backdropRef.current) {
			backdropRef.current.style.transition = "opacity 0.4s ease";
			backdropRef.current.style.opacity = "1";
		}
	}

	// Entry animation
	useEffect(() => {
		const sheet = sheetRef.current;
		const backdrop = backdropRef.current;
		if (!sheet || !backdrop) return;

		sheet.style.transition = "none";
		sheet.style.transform = "translateY(100%)";
		backdrop.style.transition = "none";
		backdrop.style.opacity = "0";

		void sheet.offsetHeight;

		requestAnimationFrame(() => {
			sheet.style.transition = "transform 0.4s cubic-bezier(0.32, 0.72, 0, 1)";
			sheet.style.transform = "translateY(0)";
			backdrop.style.transition = "opacity 0.3s ease";
			backdrop.style.opacity = "1";
		});
	}, []);

	function endDrag(e: PointerEvent) {
		isDragging.current = false;

		const elapsed = (performance.now() - dragStartTime.current) / 1000;
		const velocity =
			Math.abs(e.clientY - dragStartY.current) / Math.max(elapsed, 0.01) / 1000;
		const shouldClose =
			translateY.current > CLOSE_THRESHOLD || velocity > FLING_VELOCITY;

		if (shouldClose) {
			closeSheet();
		} else {
			snapBack();
		}
	}

	function moveDrag(e: PointerEvent) {
		if (!isDragging.current) return;
		const delta = e.clientY - dragStartY.current;
		if (delta < 0) return;
		translateY.current = delta;
		if (sheetRef.current) {
			sheetRef.current.style.transform = `translateY(${translateY.current}px)`;
		}
		if (backdropRef.current) {
			const opacity = Math.max(
				0,
				1 - translateY.current / (window.innerHeight * 0.4),
			);
			backdropRef.current.style.opacity = String(opacity);
		}
	}

	useEffect(() => {
		const handle = handleRef.current;
		if (!handle) return;

		function onPointerDown(e: PointerEvent) {
			isDragging.current = true;
			dragStartY.current = e.clientY;
			dragStartTime.current = performance.now();
			translateY.current = 0;
			if (sheetRef.current) sheetRef.current.style.transition = "none";
			if (backdropRef.current) backdropRef.current.style.transition = "none";
			(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		}

		handle.addEventListener("pointerdown", onPointerDown);
		handle.addEventListener("pointermove", moveDrag);
		handle.addEventListener("pointerup", endDrag);

		return () => {
			handle.removeEventListener("pointerdown", onPointerDown);
			handle.removeEventListener("pointermove", moveDrag);
			handle.removeEventListener("pointerup", endDrag);
		};
	}, []);

	useEffect(() => {
		const content = contentRef.current;
		if (!content) return;

		function onPointerDown(e: PointerEvent) {
			const target = e.target as HTMLElement;
			if (
				target.closest(
					"button, input, a, select, textarea, form, [role='slider'], [role='button'], [role='link'], [class*='touch-none']",
				)
			)
				return;
			if (content && content.scrollTop > 0) return;

			isDragging.current = true;
			dragStartY.current = e.clientY;
			dragStartTime.current = performance.now();
			translateY.current = 0;
			if (sheetRef.current) sheetRef.current.style.transition = "none";
			if (backdropRef.current) backdropRef.current.style.transition = "none";
			content?.setPointerCapture(e.pointerId);
		}

		content.addEventListener("pointerdown", onPointerDown);
		content.addEventListener("pointermove", moveDrag);
		content.addEventListener("pointerup", endDrag);

		return () => {
			content.removeEventListener("pointerdown", onPointerDown);
			content.removeEventListener("pointermove", moveDrag);
			content.removeEventListener("pointerup", endDrag);
		};
	}, []);

	return (
		<div class="fixed inset-0 z-50 md:hidden">
			<button
				type="button"
				ref={backdropRef}
				class="absolute inset-0 cursor-default bg-black/60 border-0 p-0"
				onClick={closeSheet}
				aria-label="Close"
			/>
			<div
				ref={sheetRef}
				class="absolute bottom-0 left-0 right-0 bg-stone-950 rounded-t-2xl flex flex-col max-h-[85vh]"
			>
				<button
					type="button"
					ref={handleRef}
					onClick={closeSheet}
					class="w-full flex items-center justify-center gap-1 pt-4 pb-2 cursor-grab active:cursor-grabbing touch-none text-white/60 hover:text-white transition-colors"
				>
					<ChevronDown size={18} />
				</button>
				<div
					ref={contentRef}
					class="overflow-y-auto min-h-0 flex-1 select-none px-4 pb-8"
					style="touch-action: pan-y; -webkit-touch-callout: none"
				>
					{children}
				</div>
			</div>
		</div>
	);
}

export function Player() {
	const player = usePlayer();

	if (!player) {
		return playerSkeletonOn.value ? <PlayerSkeleton /> : null;
	}

	const {
		song,
		cover,
		isLoading,
		isPlaying,
		toggleSong,
		progress,
		duration,
		seek,
		prevSong,
		nextSong,
		canPrev,
		canNext,
	} = player;

	const dragTracking = useRef({ startY: 0, active: false });

	const onBarPointerDown = (e: PointerEvent) => {
		const target = e.target as HTMLElement;
		if (target.closest("button, input, a, [role='slider']")) return;
		dragTracking.current.active = true;
		dragTracking.current.startY = e.clientY;
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	};

	const onBarPointerMove = (e: PointerEvent) => {
		if (!dragTracking.current.active) return;
		const delta = dragTracking.current.startY - e.clientY;
		if (delta > 30) {
			dragTracking.current.active = false;
			playerModalOpen.value = true;
		}
	};

	const onBarPointerUp = () => {
		dragTracking.current.active = false;
	};

	return (
		<>
			<div class="hidden md:block">
				<FullPlayerContent {...player} />
			</div>

			<div
				class="block md:hidden fixed bottom-0 left-0 right-0 z-40 bg-stone-950 border-t border-white/10 transition-transform duration-300 ease-out"
				style={`transform: translateY(${playerModalOpen.value ? "100%" : "0"})`}
				onPointerDown={onBarPointerDown}
				onPointerMove={onBarPointerMove}
				onPointerUp={onBarPointerUp}
			>
				<div class="px-0 pt-1 text-(--dominant-color)">
					<Progress
						value={progress}
						max={duration}
						min={0}
						onChange={seek}
						class="h-0.75 w-full rounded-none border-0 bg-neutral-800"
					/>
				</div>

				<div class="flex items-center gap-2 w-full px-4 py-3">
					<div
						role="toolbar"
						tabIndex={-1}
						onClick={() => {
							playerModalOpen.value = true;
						}}
						onKeyDown={(e) => {
							if (e.key === "Enter" || e.key === " ") {
								playerModalOpen.value = true;
							}
						}}
						class="flex items-center gap-3 flex-1 min-w-0 cursor-pointer"
					>
						<div class="relative w-12 h-12 shrink-0">
							<img
								src={cover}
								alt={song.name}
								draggable={false}
								class="w-full h-full object-cover rounded-lg shadow-lg select-none"
								style="-webkit-user-drag: none"
							/>
							<div class="absolute inset-0 rounded-lg ring-1 ring-inset ring-white/10" />
						</div>

						<div class="flex-1 min-w-0 text-left">
							<p class="text-white text-sm font-medium truncate">{song.name}</p>
							<p class="text-white/60 text-xs truncate">
								<ArtistLinks artists={song.artists} />
							</p>
						</div>
					</div>

					<div class="flex items-center gap-1 shrink-0">
						<button
							type="button"
							disabled={!canPrev}
							onClick={() => void prevSong()}
							class="p-1.5 rounded-full transition disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer text-white/80"
						>
							<SkipBack size={20} class="fill-current" />
						</button>

						<Toggle
							loading={isLoading}
							playing={isPlaying}
							onClick={() => void toggleSong()}
							size={22}
							class="bg-(--dominant-color)"
						/>

						<button
							type="button"
							disabled={!canNext}
							onClick={() => void nextSong()}
							class="p-1.5 rounded-full transition disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer text-white/80"
						>
							<SkipForward size={20} class="fill-current" />
						</button>

						<button
							type="button"
							aria-label="Expand player"
							onClick={(e) => {
								e.stopPropagation();
								playerModalOpen.value = true;
							}}
							class="shrink-0 p-1.5 rounded-full text-white/60 hover:text-white cursor-pointer transition-colors"
						>
							<ChevronUp size={18} />
						</button>
					</div>
				</div>
			</div>

			{playerModalOpen.value && (
				<DraggableSheet
					onClose={() => {
						playerModalOpen.value = false;
					}}
				>
					<div class="pt-2">
						<FullPlayerContent {...player} />
					</div>
				</DraggableSheet>
			)}
		</>
	);
}
