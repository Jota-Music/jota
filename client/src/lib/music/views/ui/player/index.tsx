import { signal } from "@preact/signals";
import { ListMusic, SkipBack, SkipForward } from "lucide-preact";
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
					class="text-current mx-auto w-52 drop-shadow-lg drop-shadow-black"
				>
					<div class="relative aspect-square w-full">
						<img
							src={cover}
							alt={song.name}
							class="absolute inset-0 h-full w-full object-cover rounded-2xl shadow-lg opacity-40 animate-[spin_120s_linear_infinite]"
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
					{song.album.title}
				</small>

				<h1 class="text-white text-xl font-semibold truncate">{song.name}</h1>

				<p class="text-white text-sm opacity-70 truncate">
					{song.artists.map((a) => a.name).join(", ")}
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
	const backdropRef = useRef<HTMLDivElement>(null);
	const handleRef = useRef<HTMLDivElement>(null);
	const dragStartY = useRef(0);
	const isDragging = useRef(false);
	const translateY = useRef(0);
	const onCloseRef = useRef(onClose);
	onCloseRef.current = onClose;

	const CLOSE_THRESHOLD = 120;

	useEffect(() => {
		const handle = handleRef.current;
		if (!handle) return;

		function onPointerDown(e: PointerEvent) {
			isDragging.current = true;
			dragStartY.current = e.clientY;
			translateY.current = 0;
			if (sheetRef.current) sheetRef.current.style.transition = "none";
			if (backdropRef.current) backdropRef.current.style.transition = "none";
			(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		}

		function onPointerMove(e: PointerEvent) {
			if (!isDragging.current) return;
			const delta = (e.clientY - dragStartY.current) * 0.5;
			translateY.current = Math.max(0, delta);
			if (sheetRef.current) {
				sheetRef.current.style.transform = `translateY(${translateY.current}px)`;
			}
			if (backdropRef.current) {
				const opacity = Math.max(
					0,
					1 - translateY.current / (window.innerHeight * 0.35),
				);
				backdropRef.current.style.opacity = String(opacity);
			}
		}

		function onPointerUp() {
			if (!isDragging.current) return;
			isDragging.current = false;

			if (translateY.current > CLOSE_THRESHOLD) {
				if (sheetRef.current) {
					sheetRef.current.style.transition = "transform 0.25s ease";
					sheetRef.current.style.transform = "translateY(100%)";
				}
				if (backdropRef.current) {
					backdropRef.current.style.transition = "opacity 0.25s ease";
					backdropRef.current.style.opacity = "0";
				}
				setTimeout(() => {
					onCloseRef.current();
					translateY.current = 0;
				}, 250);
			} else {
				if (sheetRef.current) {
					sheetRef.current.style.transition =
						"transform 0.35s cubic-bezier(0.32, 0.72, 0, 1)";
					sheetRef.current.style.transform = "translateY(0)";
				}
				if (backdropRef.current) {
					backdropRef.current.style.transition = "opacity 0.35s ease";
					backdropRef.current.style.opacity = "1";
				}
			}
		}

		handle.addEventListener("pointerdown", onPointerDown);
		handle.addEventListener("pointermove", onPointerMove);
		handle.addEventListener("pointerup", onPointerUp);

		return () => {
			handle.removeEventListener("pointerdown", onPointerDown);
			handle.removeEventListener("pointermove", onPointerMove);
			handle.removeEventListener("pointerup", onPointerUp);
		};
	}, []);

	return (
		<div class="fixed inset-0 z-50 md:hidden">
			<div
				ref={backdropRef}
				class="absolute inset-0 bg-black/60"
				onClick={onClose}
			/>
			<div
				ref={sheetRef}
				class="absolute bottom-0 left-0 right-0 bg-stone-950 rounded-t-2xl flex flex-col max-h-[85vh]"
			>
				<div
					ref={handleRef}
					class="w-full flex justify-center pt-3 pb-1 cursor-grab active:cursor-grabbing touch-none"
				>
					<div class="w-10 h-1 bg-white/20 rounded-full" />
				</div>
				<div class="overflow-y-auto min-h-0 flex-1 px-4 pb-8">
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

	const { song, cover, isLoading, isPlaying, toggleSong } = player;

	return (
		<>
			<div class="hidden md:block">
				<FullPlayerContent {...player} />
			</div>

			{!playerModalOpen.value && (
				<div class="block md:hidden fixed bottom-0 left-0 right-0 z-40 bg-stone-950 border-t border-white/10">
					<button
						type="button"
						onClick={() => {
							playerModalOpen.value = true;
						}}
						class="flex items-center gap-3 w-full px-4 py-3 cursor-pointer"
					>
						<div class="relative w-12 h-12 shrink-0">
							<img
								src={cover}
								alt={song.name}
								class="w-full h-full object-cover rounded-lg shadow-lg"
							/>
							<div class="absolute inset-0 rounded-lg ring-1 ring-inset ring-white/10" />
						</div>

						<div class="flex-1 min-w-0 text-left">
							<p class="text-white text-sm font-medium truncate">
								{song.name}
							</p>
							<p class="text-white/60 text-xs truncate">
								{song.artists.map((a) => a.name).join(", ")}
							</p>
						</div>

						<div onClick={(e) => e.stopPropagation()}>
							<Toggle
								loading={isLoading}
								playing={isPlaying}
								onClick={() => void toggleSong()}
								size={22}
								class="bg-(--dominant-color) shrink-0"
							/>
						</div>
					</button>
				</div>
			)}

			{playerModalOpen.value && (
				<DraggableSheet onClose={() => { playerModalOpen.value = false; }}>
					<div class="pt-2">
						<FullPlayerContent {...player} />
					</div>
				</DraggableSheet>
			)}
		</>
	);
}
