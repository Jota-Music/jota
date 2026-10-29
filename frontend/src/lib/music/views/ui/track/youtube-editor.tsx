import type { Video } from "@models/services/youtube/models";
import { useSignal } from "@preact/signals";
import { useQuery } from "@tanstack/preact-query";
import { Check, Link2, Loader2, Pause, Play, Search, X } from "lucide-preact";
import { useEffect, useLayoutEffect, useRef } from "preact/hooks";
import { getYouTubeId, updateYoutubeId } from "@/lib/music/app/get-audio";
import { searchYouTube, youtubeVideoToSong } from "@/lib/music/app/search";
import { parseYoutubeLink } from "@/lib/music/app/youtube-link";
import type { Song } from "@/lib/music/model";
import { coarse } from "@/lib/music/views/hooks/row-select";
import { currentSong, setYoutube } from "@/lib/music/views/stores/audio";
import { AudioCache } from "@/lib/music/views/stores/cache";
import {
	playerModalOpen,
	reloadCurrent,
} from "@/lib/music/views/stores/player";
import {
	preview,
	previewDuration,
	previewing,
	previewPending,
	previewPlaying,
	previewProgress,
	previewSeeking,
	seekPreview,
	stopPreview,
	togglePreview,
} from "@/lib/music/views/stores/preview";
import {
	editingSong,
	editorMode,
} from "@/lib/music/views/stores/youtube-editor";
import TrackArt from "@/lib/music/views/ui/track/track-art";
import VolumeControl from "@/lib/music/views/ui/volume";
import { t } from "@/lib/shared/i18n";
import { secondsToTime } from "@/lib/shared/utils/format";
import { cn } from "@/lib/shared/utils/tw";
import { addError } from "@/lib/shared/views/stores/errors";
import { Loading } from "@/lib/shared/views/ui/components/loading";
import { Modal, ModalHeader } from "@/lib/shared/views/ui/components/modal";
import Progress from "@/lib/shared/views/ui/components/progress";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

type Mode = "assign" | "search";

export default function YoutubeEditor() {
	const activeSong = editingSong.value;
	const closingSong = useSignal<Song | null>(null);
	const open = activeSong != null;
	const song = activeSong ?? closingSong.value;
	const youtubeId = useSignal("");
	const search = useSignal("");
	const query = useSignal("");
	const resolving = useSignal(false);
	const $input = useRef<HTMLInputElement>(null);
	const $results = useRef<HTMLDivElement>(null);
	const lastTap = useRef<{ id: string; time: number } | null>(null);
	const timer = useRef<number | undefined>(undefined);

	useLayoutEffect(() => {
		if (!open || !song) return;
		closingSong.value = song;
		playerModalOpen.value = false;
		stopPreview();
		lastTap.current = null;
		editorMode.value = "assign";
		youtubeId.value = song.youtubeId ?? "";
		search.value = "";
		query.value = "";
		let active = true;
		let focusFrame = 0;
		if (!coarse) {
			focusFrame = requestAnimationFrame(() => {
				focusFrame = requestAnimationFrame(() => {
					if (!active) return;
					$input.current?.focus({ preventScroll: true });
					$input.current?.select();
				});
			});
		}

		if (!song.youtubeId) {
			getYouTubeId(song.id)
				.then((id) => {
					if (active) youtubeId.value = id;
				})
				.catch(() => {});
		}
		return () => {
			active = false;
			cancelAnimationFrame(focusFrame);
			window.clearTimeout(timer.current);
			resolving.value = false;
			stopPreview();
		};
	}, [open, song]);

	useEffect(() => {
		const value = youtubeId.value.trim();
		const parsed = parseYoutubeLink(value.replace(/^youtube:/i, ""));

		if (parsed?.type !== "video" || value === parsed.id) {
			resolving.value = false;
			return;
		}

		resolving.value = true;
		window.clearTimeout(timer.current);

		timer.current = window.setTimeout(() => {
			youtubeId.value = parsed.id;
			resolving.value = false;
		}, 300);

		return () => window.clearTimeout(timer.current);
	}, [youtubeId.value]);

	useEffect(() => {
		if (!open || editorMode.value !== "search") return;
		const value = search.value.trim();
		if (value === query.value) return;
		const id = window.setTimeout(() => {
			query.value = value;
		}, 350);
		return () => window.clearTimeout(id);
	}, [open, editorMode.value, search.value]);

	useEffect(() => {
		stopPreview();
	}, [query.value]);

	const videosQuery = useQuery({
		queryKey: ["youtube-search", query.value],
		queryFn: () => searchYouTube(query.value),
		enabled: open && editorMode.value === "search" && query.value.length > 0,
		gcTime: 0,
	});
	const videos: Video[] = videosQuery.data ?? [];
	const searching = editorMode.value === "search";
	const changed = song ? youtubeId.value !== (song.youtubeId ?? "") : false;

	function switchMode(next: Mode) {
		if (next === editorMode.value) {
			$input.current?.focus();
			return;
		}

		editorMode.value = next;
		if (next === "assign") stopPreview();
		if (next === "search" && song) {
			const value = search.value.trim();
			if (value) {
				query.value = value;
			} else {
				const terms = [
					song.name,
					...(song.artists ?? []).map((artist) => artist.name),
				].join(" ");
				search.value = terms;
				query.value = terms;
			}
		}

		requestAnimationFrame(() => {
			$input.current?.focus();
			$input.current?.select();
		});
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!song) return;
		const raw = youtubeId.value.trim().replace(/^youtube:/i, "");
		const parsed = parseYoutubeLink(raw);

		if (raw && parsed?.type !== "video") {
			addError(t("music.youtubeId.invalid"));
			return;
		}

		const youtube = parsed?.type === "video" ? parsed.id : "";

		if (song.youtubeId === youtube) {
			editingSong.value = null;
			return;
		}

		try {
			await updateYoutubeId(song.id, youtube);
		} catch (error) {
			addError(error, t("music.youtubeId.title"));
			return;
		}

		setYoutube(song, youtube);
		youtubeId.value = youtube;
		AudioCache.remove(song.id);

		if (currentSong.value?.id === song.id) {
			await reloadCurrent();
		}

		editingSong.value = null;
	}

	return (
		<Modal
			open={open}
			close={() => (editingSong.value = null)}
			labelledBy="youtube-id-title"
		>
			{song && (
				<form class="flex min-h-0 flex-1 flex-col" onSubmit={handleSubmit}>
					<ModalHeader>
						<div class="flex min-w-0 flex-col">
							<h2
								id="youtube-id-title"
								class="text-sm font-semibold text-white"
							>
								{t("music.youtubeId.title")}
							</h2>
							<p class="mt-0.5 truncate text-xs text-zinc-500">{song.name}</p>
						</div>
					</ModalHeader>

					<div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4 sm:p-5">
						<p class="shrink-0 text-balance text-xs text-zinc-500">
							{t(
								searching
									? "music.youtubeId.hintSearch"
									: "music.youtubeId.hintAssign",
							)}
						</p>

						<div class="flex h-10 shrink-0 items-stretch overflow-hidden rounded-md border border-zinc-800 bg-zinc-950 focus-within:border-zinc-600">
							<div class="flex shrink-0 items-stretch border-r border-zinc-800">
								<button
									type="button"
									title={t("music.youtubeId.assignMode")}
									aria-label={t("music.youtubeId.assignMode")}
									aria-pressed={!searching}
									onClick={() => switchMode("assign")}
									class={cn(
										"flex w-10 cursor-pointer items-center justify-center transition-colors",
										!searching
											? "bg-zinc-800 text-white"
											: "text-zinc-500 hover:text-zinc-300",
									)}
								>
									<Link2 class="size-4" strokeWidth={2.5} />
								</button>
								<button
									type="button"
									title={t("music.youtubeId.searchMode")}
									aria-label={t("music.youtubeId.searchMode")}
									aria-pressed={searching}
									onClick={() => switchMode("search")}
									class={cn(
										"flex w-10 cursor-pointer items-center justify-center border-l border-zinc-800 transition-colors",
										searching
											? "bg-zinc-800 text-white"
											: "text-zinc-500 hover:text-zinc-300",
									)}
								>
									<Search class="size-4" strokeWidth={2.5} />
								</button>
							</div>

							{!searching && resolving.value && (
								<Loader2 class="ml-3 size-4 shrink-0 animate-spin text-white/30" />
							)}

							<input
								ref={$input}
								type="text"
								value={searching ? search.value : youtubeId.value}
								placeholder={t(
									searching
										? "music.youtubeId.searchPlaceholder"
										: "music.youtubeId.placeholder",
								)}
								aria-label={t(
									searching
										? "music.youtubeId.search"
										: "music.youtubeId.placeholder",
								)}
								aria-busy={!searching && resolving.value}
								class="h-full min-w-0 flex-1 bg-transparent px-3 text-sm text-white outline-none placeholder:text-zinc-600"
								onInput={(e) => {
									const value = (e.target as HTMLInputElement).value;
									if (searching) search.value = value;
									else youtubeId.value = value;
								}}
								onKeyDown={(e) => {
									if (e.key === "Enter" && searching) {
										e.preventDefault();
										query.value = search.value.trim();
									}
									if (e.key === "Escape") {
										editingSong.value = null;
									}
								}}
							/>

							{(searching ? search.value : youtubeId.value) && (
								<button
									type="button"
									title={t("search.clear")}
									aria-label={t("search.clear")}
									onClick={() => {
										if (searching) {
											search.value = "";
											query.value = "";
										} else {
											youtubeId.value = "";
										}
									}}
									class="mr-2 flex size-6 shrink-0 cursor-pointer items-center justify-center self-center rounded text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-white"
								>
									<X size={14} />
								</button>
							)}
						</div>

						<div
							class={cn(
								"grid min-h-0 transition-[grid-template-rows] duration-200",
								searching ? "flex-1 grid-rows-[1fr]" : "grid-rows-[0fr]",
							)}
							aria-hidden={!searching}
							inert={!searching}
						>
							<div
								class={cn(
									"relative min-h-0 overflow-hidden transition-[height,opacity] duration-200 ease-out motion-reduce:transition-none",
									searching
										? "h-64 max-h-[40dvh] opacity-100"
										: "h-0 opacity-0",
								)}
							>
								<div
									ref={$results}
									class="absolute inset-0 overflow-y-auto overscroll-contain rounded-lg border border-zinc-800 bg-zinc-950 pr-[15px]"
								>
									{query.value.length === 0 ? (
										<p class="py-6 text-sm text-zinc-500">
											{t("music.youtubeId.searchPrompt")}
										</p>
									) : videosQuery.isLoading ? (
										<Loading />
									) : videosQuery.isError ? (
										<p class="text-sm text-red-400">
											{t("music.youtubeId.searchFailed")}
										</p>
									) : videos.length === 0 ? (
										<p class="text-sm text-zinc-500">
											{t("music.youtubeId.searchEmpty")}
										</p>
									) : (
										videos.map((video) => {
											const active = video.id === youtubeId.value;
											const videoSong = youtubeVideoToSong(video);
											const current = previewing.value === videoSong.id;
											const loading = previewPending.value === videoSong.id;
											return (
												<button
													key={video.id}
													type="button"
													onClick={() => {
														const selected = youtubeId.value === video.id;
														youtubeId.value = video.id;

														if (coarse) {
															if (selected) void preview(videoSong);
															return;
														}

														const now = Date.now();
														const last = lastTap.current;
														if (
															last?.id === video.id &&
															now - last.time < 350
														) {
															lastTap.current = null;
															void preview(videoSong);
															return;
														}
														lastTap.current = { id: video.id, time: now };
													}}
													class={cn(
														"flex h-16 w-full cursor-pointer touch-manipulation select-none items-center justify-between gap-2 border-b border-zinc-900 px-3 text-left transition",
														active ? "bg-zinc-800/60" : "hover:bg-zinc-800/60",
													)}
												>
													<TrackArt
														song={videoSong}
														current={current}
														loading={loading}
														playing={current && previewPlaying.value}
														alt={video.title}
														class={
															current
																? "rounded-md outline-2 outline-(--dominant-color)"
																: undefined
														}
														imgClass="size-10 shrink-0 rounded-md object-cover"
													/>
													<div class="flex min-w-0 flex-1 flex-col text-start">
														<span
															class={cn(
																"truncate text-sm",
																current
																	? "font-medium text-(--dominant-color)"
																	: "text-white",
															)}
														>
															{video.title}
														</span>
														<span class="truncate text-xs text-zinc-400">
															{video.author}
														</span>
													</div>
													<div class="flex shrink-0 items-center gap-2">
														<div class="tabular-nums text-xs text-zinc-500">
															{video.duration
																? secondsToTime(video.duration)
																: ""}
														</div>
														{active && (
															<Check
																class="size-4 shrink-0 text-white"
																strokeWidth={3}
															/>
														)}
													</div>
												</button>
											);
										})
									)}
								</div>
								<Scrollbar target={$results} />
							</div>
						</div>
					</div>

					<footer class="shrink-0 border-t border-zinc-800">
						{previewing.value && (
							<div class="flex items-center gap-2 border-b border-zinc-800 px-3 py-2">
								<button
									type="button"
									title={
										previewPlaying.value ? t("music.pause") : t("music.play")
									}
									aria-label={
										previewPlaying.value ? t("music.pause") : t("music.play")
									}
									aria-pressed={previewPlaying.value}
									onClick={togglePreview}
									class="flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-md text-zinc-300 transition hover:bg-zinc-800 hover:text-white"
								>
									{previewPending.value ? (
										<Loader2 class="size-4 animate-spin" />
									) : previewPlaying.value ? (
										<Pause class="size-4 fill-current" />
									) : (
										<Play class="size-4 fill-current" />
									)}
								</button>

								<span class="hidden tabular-nums text-xs text-zinc-500 sm:inline">
									{secondsToTime(previewProgress.value)}
								</span>

								<Progress
									class="h-1.5 min-w-0 flex-1 bg-zinc-800 text-(--dominant-color)"
									value={previewProgress.value}
									max={previewDuration.value > 0 ? previewDuration.value : 1}
									onChange={(value) => {
										previewSeeking.value = true;
										previewProgress.value = value;
									}}
									onCommit={(value) => {
										previewSeeking.value = false;
										seekPreview(value);
									}}
									disabled={
										previewPending.value !== null || previewDuration.value <= 0
									}
								/>

								<span class="tabular-nums text-xs text-zinc-500">
									{secondsToTime(previewDuration.value)}
								</span>

								<VolumeControl compact />

								<button
									type="button"
									title={t("music.youtubeId.previewStop")}
									aria-label={t("music.youtubeId.previewStop")}
									onClick={stopPreview}
									class="flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-md text-zinc-500 transition hover:bg-zinc-800 hover:text-white"
								>
									<X class="size-4" />
								</button>
							</div>
						)}

						<div class="flex justify-end p-3">
							<button
								type="submit"
								disabled={!changed}
								title={t("music.youtubeId.save")}
								aria-label={t("music.youtubeId.save")}
								class="flex size-10 shrink-0 cursor-pointer items-center justify-center rounded-md bg-(--dominant-color) text-(--binary-color) transition-opacity hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-40"
							>
								<Check class="size-4" strokeWidth={3} />
							</button>
						</div>
					</footer>
				</form>
			)}
		</Modal>
	);
}
