import { computed, signal, useSignal } from "@preact/signals";
import {
	ChevronDown,
	ChevronUp,
	Minimize2,
	Repeat,
	Repeat1,
	Shuffle,
	SkipBack,
	SkipForward,
} from "lucide-preact";
import { memo } from "preact/compat";
import { useEffect, useLayoutEffect, useRef } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { usePlayer } from "@/lib/music/views/hooks/use-player";
import { audioDuration, progress } from "@/lib/music/views/stores/audio";
import {
	commitSeek,
	playerModalOpen,
	previewSeek,
} from "@/lib/music/views/stores/player";
import {
	canNext,
	cycleRepeat,
	repeat,
	showQueue,
	shuffle,
	toggleShuffle,
} from "@/lib/music/views/stores/queue";
import QueueButton from "@/lib/music/views/ui/player/queue-button";
import SaveYoutubeId from "@/lib/music/views/ui/player/save-youtube-id";
import Toggle from "@/lib/music/views/ui/player/toggle";
import TrackActions, {
	useTrackMenu,
} from "@/lib/music/views/ui/track/track-actions";
import VolumeControl from "@/lib/music/views/ui/volume";
import { t } from "@/lib/shared/i18n";
import { cover as coverImage } from "@/lib/shared/utils/cover";
import { secondsToTime } from "@/lib/shared/utils/format";
import { isAndroid } from "@/lib/shared/utils/platform";
import { cn } from "@/lib/shared/utils/tw";
import AlbumLink from "@/lib/shared/views/ui/components/album-link";
import ArtistLinks from "@/lib/shared/views/ui/components/artist-links";
import CircularProgress from "@/lib/shared/views/ui/components/circular-progress";
import { closeContextMenu } from "@/lib/shared/views/ui/components/context-menu";
import { CoverFallback } from "@/lib/shared/views/ui/components/cover-fallback";
import { Image } from "@/lib/shared/views/ui/components/image";
import { Modal } from "@/lib/shared/views/ui/components/modal";
import Progress from "@/lib/shared/views/ui/components/progress";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

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
	// On the phone the sheet sits over the compact bar, which already carries a
	// seek line: the sheet keeps the controls and closes with the times, which
	// then read as that line's labels.
	narrow?: boolean;
}

const Disc = memo(function Disc({
	cover,
	disabled = false,
	isLoading,
	isPlaying,
	narrow,
	toggleSong,
	nextSong,
	prevSong,
}: {
	cover?: string;
	disabled?: boolean;
	isLoading: boolean;
	isPlaying: boolean;
	narrow?: boolean;
	toggleSong: () => Promise<void>;
	nextSong: () => Promise<void>;
	prevSong: () => Promise<void>;
}) {
	return (
		<CircularProgress
			value={progress.value}
			max={audioDuration.value}
			min={0}
			onChange={previewSeek}
			onCommit={commitSeek}
			disabled={disabled}
			class={cn(
				"text-current mx-auto",
				// The sheet is capped by the space above the player bar, so a
				// fixed disc does not fit once the keyboard or a short window
				// eats that space. Keep the design size until the viewport is
				// too short for it.
				narrow ? "w-[min(12rem,36dvh)]" : "w-48",
			)}
		>
			<div class="relative aspect-square w-full">
				<Image
					src={coverImage(cover, 384)}
					alt=""
					loading="eager"
					draggable={false}
					class={cn(
						"absolute inset-0 h-full w-full object-cover rounded-2xl shadow-lg opacity-40 animate-[spin_120s_linear_infinite] select-none",
						!isPlaying && "[animation-play-state:paused]",
					)}
					style="-webkit-user-drag: none"
					fallback={<CoverFallback class="absolute inset-0 rounded-full" />}
				/>

				<div
					class={cn(
						"absolute inset-0 w-max h-max flex items-center justify-center m-auto gap-6",
						narrow && "[@media(max-height:533px)]:gap-2",
					)}
				>
					<button
						type="button"
						disabled={disabled}
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
						disabled={disabled || !canNext.value}
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
	const { song, isLoading, isPlaying, toggleSong, narrow } = props;

	return (
		<section
			class={cn(
				"w-full max-w-2xl mx-auto relative z-10 flex flex-col md:flex-row gap-4 md:gap-6 items-start justify-center text-(--dominant-color) before:pointer-events-none before:absolute before:-inset-12 before:-z-10 before:rounded-[3rem] before:bg-[radial-gradient(closest-side,rgba(12,10,9,0.94),rgba(12,10,9,0))] before:content-['']",
				narrow && "before:hidden",
			)}
		>
			<div class="relative flex max-h-max w-full md:w-max">
				<Disc {...props} />
			</div>

			<div class="flex-1 min-w-0 w-full">
				<small class="text-white text-xs opacity-60 block truncate">
					<AlbumLink album={song.album} playlistId={song.playlistId} />
				</small>

				<h1 class="text-white text-xl font-semibold truncate">{song.name}</h1>

				<p class="text-white text-sm opacity-70 truncate">
					<ArtistLinks artists={song.artists} />
				</p>

				<SaveYoutubeId song={song} />

				{narrow ? (
					<>
						<PlayerControlsRow
							song={song}
							isLoading={isLoading}
							isPlaying={isPlaying}
							toggleSong={toggleSong}
						/>

						<div class="flex items-center justify-between">
							<CurrentTime />

							<TotalTime />
						</div>
					</>
				) : (
					<>
						<div class="mt-3 flex gap-2 justify-between items-center">
							<CurrentTime />

							<SeekBar class="h-1.5 min-w-0 flex-1 opacity-80" />

							<TotalTime />
						</div>

						<PlayerControlsRow
							song={song}
							isLoading={isLoading}
							isPlaying={isPlaying}
							toggleSong={toggleSong}
						/>
					</>
				)}
			</div>
		</section>
	);
}

type ControlsRowProps = {
	disabled?: boolean;
	song?: Song;
	isLoading: boolean;
	isPlaying: boolean;
	toggleSong: () => Promise<void>;
};

function PlayerControlsRow({
	disabled = false,
	song,
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
							: "text-neutral-400 hover:text-white",
					)}
				>
					<Shuffle class="size-5.5 stroke-current" />
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
							: "text-neutral-400 hover:text-white",
					)}
				>
					{repeat.value === "one" ? (
						<Repeat1 class="size-5.5 stroke-current" />
					) : (
						<Repeat class="size-5.5 stroke-current" />
					)}
				</button>
			</div>

			<Toggle
				loading={isLoading}
				playing={isPlaying}
				onClick={() => void toggleSong()}
				disabled={disabled}
				iconClass="size-6"
				class={cn(
					"max-md:hidden",
					disabled ? "bg-neutral-700" : "bg-(--dominant-color)",
				)}
			/>

			<VolumeControl class="max-md:flex-1" />

			<QueueButton iconClass="size-5.5" />

			{song && (
				<div class="max-md:hidden">
					<TrackActions songs={[song]} onPlay={() => void toggleSong()} />
				</div>
			)}

			<button
				type="button"
				title={t("music.player.compact")}
				onClick={() => setCompactPlayer(true)}
				class="hidden md:block rounded-md p-1 transition text-neutral-400 hover:text-white cursor-pointer"
			>
				<Minimize2 class="size-6" />
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

function NothingPlayingContent({ narrow = false }: { narrow?: boolean }) {
	return (
		<section
			class={cn(
				"w-full max-w-2xl mx-auto relative z-10 flex flex-col md:flex-row gap-4 md:gap-6 items-start justify-center text-neutral-500 before:pointer-events-none before:absolute before:-inset-12 before:-z-10 before:rounded-[3rem] before:bg-[radial-gradient(closest-side,rgba(12,10,9,0.94),rgba(12,10,9,0))] before:content-['']",
				narrow && "before:hidden",
			)}
		>
			<div class="relative flex max-h-max w-full md:w-max">
				<Disc
					disabled
					narrow={narrow}
					isLoading={false}
					isPlaying={false}
					toggleSong={noop}
					nextSong={noop}
					prevSong={noop}
				/>
			</div>
			<div class="flex-1 min-w-0 w-full">
				<h1 class="text-neutral-400 text-xl font-semibold truncate">
					{t("music.player.nothingPlaying")}
				</h1>
				{narrow ? (
					<>
						<PlayerControlsRow
							disabled
							isLoading={false}
							isPlaying={false}
							toggleSong={noop}
						/>
						<div class="flex items-center justify-between">
							<CurrentTime />
							<TotalTime />
						</div>
					</>
				) : (
					<>
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
					</>
				)}
			</div>
		</section>
	);
}

export function Player() {
	const player = usePlayer();
	const onContextMenu = useTrackMenu(
		player?.song,
		() => void player?.toggleSong(),
	);

	return (
		<div
			role="none"
			class={compactPlayer.value ? "hidden" : "hidden md:block"}
			onContextMenu={onContextMenu}
		>
			{player ? <FullPlayerContent {...player} /> : <NothingPlayingContent />}
		</div>
	);
}

export function PlayerBar() {
	const player = usePlayer();
	const hasSong = player != null;
	const isLoading = player?.isLoading ?? false;
	const isPlaying = player?.isPlaying ?? false;

	const dragTracking = useRef({ startY: 0, active: false });
	const modalListRef = useRef<HTMLDivElement>(null);
	const barRef = useRef<HTMLDivElement>(null);
	const barHeight = useSignal(0);

	useLayoutEffect(() => {
		if (barRef.current) barHeight.value = barRef.current.offsetHeight;
	}, []);

	// Opened from the compact bar, so clear the bar itself: the menu would
	// otherwise open on top of the controls it sits next to.
	const onContextMenu = useTrackMenu(
		player?.song,
		() => void player?.toggleSong(),
		barHeight.value,
		true,
	);

	// Expanded player (mobile modal): the compact bar is slid out of view.
	const onExpandedMenu = useTrackMenu(
		player?.song,
		() => void player?.toggleSong(),
	);

	useEffect(() => {
		if (showQueue.value) playerModalOpen.value = false;
	}, [showQueue.value]);

	useEffect(() => {
		if (playerModalOpen.value) closeContextMenu();
	}, [playerModalOpen.value]);

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
			playerModalOpen.value = !playerModalOpen.value;
		}
	};

	return (
		<>
			<div
				ref={barRef}
				role="none"
				class={cn(
					"block w-full z-60 md:z-40 bg-stone-950 border-t border-white/10",
					compactPlayer.value ? "md:block" : "md:hidden",
				)}
				onContextMenu={onContextMenu}
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
						class="h-0.75 w-full rounded-none border-0 [&>div:first-child]:rounded-none"
						disabled={!hasSong}
					/>
				</div>

				<div class="flex items-center gap-2 w-full px-4 py-3 max-[420px]:gap-1 max-[420px]:px-2">
					<div
						role="toolbar"
						tabIndex={-1}
						onClick={onExpand}
						onKeyDown={(e) => {
							if (e.key === "Enter" || e.key === " ") {
								onExpand();
							}
						}}
						class="flex items-center gap-3 flex-1 min-w-0 cursor-pointer max-[420px]:gap-2"
					>
						{hasSong ? (
							<>
								<div class="relative w-12 h-12 shrink-0 max-[420px]:w-10 max-[420px]:h-10">
									<Image
										src={coverImage(player.cover, 128)}
										alt={player.song.name}
										loading="eager"
										draggable={false}
										class="w-full h-full object-cover rounded-lg shadow-lg select-none"
										style="-webkit-user-drag: none"
										fallback={
											<CoverFallback class="h-full w-full rounded-lg" />
										}
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

					<div class="flex items-center gap-1 md:gap-3 shrink-0">
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
										: "text-neutral-400 hover:text-white",
								)}
							>
								<Shuffle class="size-5.5 stroke-current" />
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
										: "text-neutral-400 hover:text-white",
								)}
							>
								{repeat.value === "one" ? (
									<Repeat1 class="size-5.5 stroke-current" />
								) : (
									<Repeat class="size-5.5 stroke-current" />
								)}
							</button>
						</div>

						<div class="flex items-center gap-1 max-[420px]:gap-0.5">
							{/* Android has no room to spare here and its menu carries the
							    switch instead; elsewhere the speaker stays in the row. */}
							{!isAndroid && <VolumeControl tip />}

							<button
								type="button"
								disabled={!hasSong}
								onClick={() => void player?.prevSong()}
								class="p-1.5 rounded-full transition disabled:cursor-not-allowed cursor-pointer text-neutral-300"
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
								disabled={!hasSong || !canNext.value}
								onClick={() => void player?.nextSong()}
								class="p-1.5 rounded-full transition disabled:cursor-not-allowed cursor-pointer text-neutral-300"
							>
								<SkipForward class="size-6 fill-current" />
							</button>
						</div>

						<div class="flex items-center gap-1 max-[420px]:gap-0.5">
							<QueueButton class="max-md:hidden" />

							{player && (
								<TrackActions
									songs={[player.song]}
									onPlay={() => void player.toggleSong()}
									bottom={barHeight.value}
									volume
								/>
							)}

							<button
								type="button"
								aria-label={t("music.player.expand")}
								onClick={(e) => {
									e.stopPropagation();
									onExpand();
								}}
								class="shrink-0 p-1.5 rounded-full text-neutral-400 hover:text-white cursor-pointer transition-colors"
							>
								{playerModalOpen.value ? (
									<ChevronDown class="size-5" />
								) : (
									<ChevronUp class="size-5" />
								)}
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
			>
				<div
					role="none"
					class="relative flex min-h-0 flex-1 flex-col"
					onContextMenu={onExpandedMenu}
				>
					<div
						ref={modalListRef}
						class="min-h-0 flex-1 overflow-y-auto px-6 pt-4 pb-4"
					>
						{hasSong ? (
							<FullPlayerContent {...player} narrow />
						) : (
							<NothingPlayingContent narrow />
						)}
					</div>
					<Scrollbar target={modalListRef} gutter={24} />
				</div>
			</Modal>
		</>
	);
}
