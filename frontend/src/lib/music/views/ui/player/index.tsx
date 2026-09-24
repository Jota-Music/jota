import { computed, signal } from "@preact/signals";
import {
	ChevronUp,
	Minimize2,
	Repeat,
	Repeat1,
	Shuffle,
	SkipBack,
	SkipForward,
} from "lucide-preact";
import { memo } from "preact/compat";
import { useEffect, useRef } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { usePlayer } from "@/lib/music/views/hooks/use-player";
import { audioDuration, progress } from "@/lib/music/views/stores/audio";
import { commitSeek, previewSeek } from "@/lib/music/views/stores/player";
import {
	cycleRepeat,
	repeat,
	showQueue,
	shuffle,
	toggleShuffle,
} from "@/lib/music/views/stores/queue";
import QueueButton from "@/lib/music/views/ui/player/queue-button";
import SaveYoutubeId from "@/lib/music/views/ui/player/save-youtube-id";
import Toggle from "@/lib/music/views/ui/player/toggle";
import VolumeControl from "@/lib/music/views/ui/volume";
import { t } from "@/lib/shared/i18n";
import { cover as coverImage } from "@/lib/shared/utils/cover";
import { secondsToTime } from "@/lib/shared/utils/format";
import { cn } from "@/lib/shared/utils/tw";
import AlbumLink from "@/lib/shared/views/ui/components/album-link";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import CircularProgress from "@/lib/shared/views/ui/components/circular-progress";
import { Image } from "@/lib/shared/views/ui/components/image";
import { Modal, ModalHeader } from "@/lib/shared/views/ui/components/modal";
import Progress from "@/lib/shared/views/ui/components/progress";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

const playerModalOpen = signal(false);

const currentLabel = computed(() => secondsToTime(progress.value));
const totalLabel = computed(() => secondsToTime(audioDuration.value));

const PLAYER_COMPACT_KEY = "music-player-compact";

export const compactPlayer = signal(
	typeof window !== "undefined" &&
		localStorage.getItem(PLAYER_COMPACT_KEY) === "1",
);

function setCompactPlayer(value: boolean) {
	compactPlayer.value = value;
	if (typeof window !== "undefined") {
		localStorage.setItem(PLAYER_COMPACT_KEY, value ? "1" : "0");
	}
}

interface FullPlayerProps {
	song: Song;
	cover: string;
	isLoading: boolean;
	isPlaying: boolean;
	toggleSong: () => Promise<void>;
	nextSong: () => Promise<void>;
	prevSong: () => Promise<void>;
	canPrev: boolean;
	canNext: boolean;
}

const Disc = memo(function Disc({
	cover,
	disabled = false,
	isLoading,
	isPlaying,
	toggleSong,
	nextSong,
	prevSong,
	canPrev,
	canNext,
}: {
	cover?: string;
	disabled?: boolean;
	isLoading: boolean;
	isPlaying: boolean;
	toggleSong: () => Promise<void>;
	nextSong: () => Promise<void>;
	prevSong: () => Promise<void>;
	canPrev: boolean;
	canNext: boolean;
}) {
	return (
		<CircularProgress
			value={progress.value}
			max={audioDuration.value}
			min={0}
			onChange={previewSeek}
			onCommit={commitSeek}
			disabled={disabled}
			class="text-current mx-auto w-48"
		>
			<div class="relative aspect-square w-full">
				{cover ? (
					<Image
						src={coverImage(cover, 384)}
						alt=""
						draggable={false}
						class={cn(
							"absolute inset-0 h-full w-full object-cover rounded-2xl shadow-lg opacity-40 animate-[spin_120s_linear_infinite] select-none",
							!isPlaying && "[animation-play-state:paused]",
						)}
						style="-webkit-user-drag: none"
					/>
				) : (
					<div class="absolute inset-0 h-full w-full rounded-full bg-stone-800" />
				)}

				<div class="absolute inset-0 w-max h-max flex items-center justify-center gap-6 m-auto">
					<button
						type="button"
						disabled={disabled || !canPrev}
						onClick={() => void prevSong()}
						class="p-1 rounded-full transition drop-shadow-lg drop-shadow-black disabled:cursor-not-allowed cursor-pointer"
					>
						<SkipBack class="size-6 fill-current shadow-lg" />
					</button>

					<Toggle
						loading={isLoading}
						playing={isPlaying}
						onClick={() => void toggleSong()}
						disabled={disabled}
						class={disabled ? "bg-neutral-700" : "bg-(--dominant-color)"}
					/>

					<button
						type="button"
						disabled={disabled || !canNext}
						onClick={() => void nextSong()}
						class="p-1 rounded-full transition drop-shadow-lg drop-shadow-black disabled:cursor-not-allowed cursor-pointer"
					>
						<SkipForward class="size-6 fill-current shadow-lg" />
					</button>
				</div>
			</div>
		</CircularProgress>
	);
});

function CurrentTime() {
	return (
		<span class="text-white text-sm opacity-70 tabular-nums shrink-0">
			{currentLabel}
		</span>
	);
}

function TotalTime() {
	return (
		<span class="text-white text-sm opacity-70 tabular-nums shrink-0">
			{totalLabel}
		</span>
	);
}

function SeekBar({
	class: className,
	disabled = false,
}: {
	class?: string;
	disabled?: boolean;
}) {
	return (
		<Progress
			value={progress.value}
			max={audioDuration.value}
			min={0}
			onChange={previewSeek}
			onCommit={commitSeek}
			class={className}
			disabled={disabled}
		/>
	);
}

function FullPlayerContent(props: FullPlayerProps) {
	const { song, isLoading, isPlaying, toggleSong } = props;

	return (
		<section class="w-full max-w-2xl mx-auto relative z-10 flex flex-col md:flex-row gap-4 md:gap-6 items-start justify-center text-(--dominant-color)">
			<div class="relative flex max-h-max w-full md:w-max">
				<Disc {...props} />
			</div>

			<div class="flex-1 min-w-0 w-full">
				<small class="text-white text-xs opacity-60 block truncate">
					<AlbumLink album={song.album} />
				</small>

				<h1 class="text-white text-xl font-semibold truncate">{song.name}</h1>

				<p class="text-white text-sm opacity-70 truncate">
					<ArtistLinks artists={song.artists} />
				</p>

				<SaveYoutubeId song={song} />

				<div class="mt-3 flex gap-2 justify-between items-center">
					<CurrentTime />

					<SeekBar class="h-1.5 min-w-0 flex-1 opacity-80" />

					<TotalTime />
				</div>

				<PlayerControlsRow
					isLoading={isLoading}
					isPlaying={isPlaying}
					toggleSong={toggleSong}
				/>
			</div>
		</section>
	);
}

type ControlsRowProps = {
	disabled?: boolean;
	isLoading: boolean;
	isPlaying: boolean;
	toggleSong: () => Promise<void>;
};

function PlayerControlsRow({
	disabled = false,
	isLoading,
	isPlaying,
	toggleSong,
}: ControlsRowProps) {
	return (
		<div class="flex items-center justify-between gap-4 mt-4 pb-2">
			<div class="flex items-center gap-2">
				<button
					type="button"
					title={
						shuffle.value
							? t("music.player.disableShuffle")
							: t("music.player.enableShuffle")
					}
					onClick={toggleShuffle}
					aria-pressed={shuffle.value}
					class={cn(
						"rounded-md p-1 transition cursor-pointer",
						shuffle.value
							? "text-(--dominant-color)"
							: "text-white/60 hover:bg-white/10 hover:text-white",
					)}
				>
					<Shuffle class="size-5 stroke-current" />
				</button>

				<button
					type="button"
					title={
						repeat.value === "off"
							? t("music.player.repeatAll")
							: repeat.value === "all"
								? t("music.player.repeatOne")
								: t("music.player.noRepeat")
					}
					onClick={cycleRepeat}
					aria-label={t("music.player.repeatMode")}
					aria-pressed={repeat.value !== "off"}
					class={cn(
						"rounded-md p-1 transition cursor-pointer",
						repeat.value !== "off"
							? "text-(--dominant-color)"
							: "text-white/60 hover:bg-white/10 hover:text-white",
					)}
				>
					{repeat.value === "one" ? (
						<Repeat1 class="size-5 stroke-current" />
					) : (
						<Repeat class="size-5 stroke-current" />
					)}
				</button>
			</div>

			<Toggle
				loading={isLoading}
				playing={isPlaying}
				onClick={() => void toggleSong()}
				disabled={disabled}
				iconClass="size-5"
				class={disabled ? "bg-neutral-700" : "bg-(--dominant-color)"}
			/>

			<VolumeControl />

			<QueueButton class="hover:bg-white/10" />

			<button
				type="button"
				title={t("music.player.compact")}
				onClick={() => setCompactPlayer(true)}
				class="hidden md:block rounded-md p-1 transition hover:bg-white/10 cursor-pointer"
			>
				<Minimize2 class="size-5" />
			</button>
		</div>
	);
}

function NothingPlaying() {
	return (
		<div class="flex-1 min-w-0 text-left">
			<p class="text-neutral-400 text-sm font-medium truncate">
				{t("music.player.nothingPlaying")}
			</p>
		</div>
	);
}

const noop = async () => {};

function NothingPlayingContent() {
	return (
		<section class="w-full max-w-2xl mx-auto relative z-10 flex flex-col md:flex-row gap-4 md:gap-6 items-start justify-center text-neutral-500">
			<div class="relative flex max-h-max w-full md:w-max">
				<Disc
					disabled
					isLoading={false}
					isPlaying={false}
					toggleSong={noop}
					nextSong={noop}
					prevSong={noop}
					canPrev={false}
					canNext={false}
				/>
			</div>
			<div class="flex-1 min-w-0 w-full">
				<h1 class="text-neutral-400 text-xl font-semibold truncate">
					{t("music.player.nothingPlaying")}
				</h1>
				<div class="mt-3 flex gap-2 justify-between items-center">
					<CurrentTime />
					<SeekBar class="h-1.5 min-w-0 flex-1 opacity-80" disabled />
					<TotalTime />
				</div>
				<PlayerControlsRow
					disabled
					isLoading={false}
					isPlaying={false}
					toggleSong={noop}
				/>
			</div>
		</section>
	);
}

export function Player() {
	const player = usePlayer();
	const hasSong = player != null;
	const isLoading = player?.isLoading ?? false;
	const isPlaying = player?.isPlaying ?? false;
	const canPrev = player?.canPrev ?? false;
	const canNext = player?.canNext ?? false;

	const dragTracking = useRef({ startY: 0, active: false });
	const modalListRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		if (showQueue.value) playerModalOpen.value = false;
	}, [showQueue.value]);

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

	const desktopViewport = () =>
		typeof window !== "undefined" &&
		window.matchMedia("(min-width: 768px)").matches;

	const onExpand = () => {
		if (compactPlayer.value && desktopViewport()) {
			setCompactPlayer(false);
		} else {
			playerModalOpen.value = true;
		}
	};

	return (
		<>
			<div class={compactPlayer.value ? "hidden" : "hidden md:block"}>
				{hasSong ? (
					<FullPlayerContent {...player} />
				) : (
					<NothingPlayingContent />
				)}
			</div>

			<div
				class={cn(
					"block fixed bottom-[calc(3.5rem+env(safe-area-inset-bottom))] md:bottom-0 left-0 right-0 z-60 md:z-40 bg-stone-950 border-t border-white/10 transition-transform duration-300 ease-out",
					compactPlayer.value ? "md:block" : "md:hidden",
				)}
				style={`transform: translateY(${
					playerModalOpen.value ? "calc(100% + 3.5rem)" : "0"
				})`}
				onPointerDown={onBarPointerDown}
				onPointerMove={onBarPointerMove}
				onPointerUp={onBarPointerUp}
			>
				<div
					class={cn(
						"px-0",
						hasSong ? "text-(--dominant-color)" : "text-neutral-600",
					)}
				>
					<SeekBar
						class="h-0.75 w-full rounded-none border-0 bg-neutral-800"
						disabled={!hasSong}
					/>
				</div>

				<div class="flex items-center gap-2 w-full px-4 py-3">
					<div
						role="toolbar"
						tabIndex={-1}
						onClick={onExpand}
						onKeyDown={(e) => {
							if (e.key === "Enter" || e.key === " ") {
								onExpand();
							}
						}}
						class="flex items-center gap-3 flex-1 min-w-0 cursor-pointer"
					>
						{hasSong ? (
							<>
								<div class="relative w-12 h-12 shrink-0">
									<Image
										src={coverImage(player.cover, 128)}
										alt={player.song.name}
										draggable={false}
										class="w-full h-full object-cover rounded-lg shadow-lg select-none"
										style="-webkit-user-drag: none"
									/>
									<div class="absolute inset-0 rounded-lg ring-1 ring-inset ring-white/10" />
								</div>

								<div class="flex-1 min-w-0 text-left">
									<p class="text-white text-sm font-medium truncate">
										{player.song.name}
									</p>
									<p class="text-white/60 text-xs truncate">
										<ArtistLinks artists={player.song.artists} />
									</p>
								</div>
							</>
						) : (
							<NothingPlaying />
						)}
					</div>

					<div class="flex items-center gap-2 md:gap-3 shrink-0">
						<div class="flex items-center gap-0.5 max-md:hidden">
							<button
								type="button"
								title={
									shuffle.value
										? t("music.player.disableShuffle")
										: t("music.player.enableShuffle")
								}
								onClick={toggleShuffle}
								aria-pressed={shuffle.value}
								class={cn(
									"rounded-md p-1.5 transition cursor-pointer",
									shuffle.value
										? "text-(--dominant-color)"
										: "text-white/60 hover:text-white",
								)}
							>
								<Shuffle class="size-5 stroke-current" />
							</button>
						</div>

						<div class="flex items-center gap-0.5 max-md:hidden">
							<button
								type="button"
								title={
									repeat.value === "off"
										? t("music.player.repeatAll")
										: repeat.value === "all"
											? t("music.player.repeatOne")
											: t("music.player.noRepeat")
								}
								onClick={cycleRepeat}
								aria-label={t("music.player.repeatMode")}
								aria-pressed={repeat.value !== "off"}
								class={cn(
									"rounded-md p-1.5 transition cursor-pointer",
									repeat.value !== "off"
										? "text-(--dominant-color)"
										: "text-white/60 hover:text-white",
								)}
							>
								{repeat.value === "one" ? (
									<Repeat1 class="size-5 stroke-current" />
								) : (
									<Repeat class="size-5 stroke-current" />
								)}
							</button>
						</div>

						<div class="flex items-center gap-1">
							<VolumeControl class="mr-2 max-[500px]:hidden" />

							<button
								type="button"
								disabled={!hasSong || !canPrev}
								onClick={() => void player?.prevSong()}
								class="p-1.5 rounded-full transition disabled:cursor-not-allowed cursor-pointer text-white/80"
							>
								<SkipBack class="size-6 fill-current" />
							</button>

							<Toggle
								loading={hasSong && isLoading}
								playing={hasSong && isPlaying}
								onClick={() => void player?.toggleSong()}
								disabled={!hasSong}
								class="bg-(--dominant-color)"
							/>

							<button
								type="button"
								disabled={!hasSong || !canNext}
								onClick={() => void player?.nextSong()}
								class="p-1.5 rounded-full transition disabled:cursor-not-allowed cursor-pointer text-white/80"
							>
								<SkipForward class="size-6 fill-current" />
							</button>
						</div>

						<div class="flex items-center gap-0.5">
							<QueueButton class="max-md:hidden" />

							<button
								type="button"
								aria-label={t("music.player.expand")}
								onClick={(e) => {
									e.stopPropagation();
									onExpand();
								}}
								class="shrink-0 p-1.5 rounded-full text-white/60 hover:text-white cursor-pointer transition-colors"
							>
								<ChevronUp class="size-5" />
							</button>
						</div>
					</div>
				</div>
			</div>

			<Modal
				open={playerModalOpen.value}
				close={() => {
					playerModalOpen.value = false;
				}}
				mobileOnly
				closeLabel={t("music.player.close")}
				hideClose
				abovePlayer={false}
			>
				<ModalHeader
					close={() => {
						playerModalOpen.value = false;
					}}
					closeLabel={t("music.player.close")}
					bordered={false}
				/>
				<div class="relative flex min-h-0 flex-1 flex-col">
					<div
						ref={modalListRef}
						class="min-h-0 flex-1 overflow-y-auto px-6 pb-4"
					>
						{hasSong ? (
							<FullPlayerContent {...player} />
						) : (
							<NothingPlayingContent />
						)}
					</div>
					<Scrollbar target={modalListRef} />
				</div>
			</Modal>
		</>
	);
}
