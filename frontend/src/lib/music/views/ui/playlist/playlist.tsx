import { effect, signal, useSignal } from "@preact/signals";
import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { ArrowUpDown, Loader, RefreshCw, Search, X } from "lucide-preact";
import { useEffect } from "preact/hooks";
import {
	getFullPlaylist,
	revalidateFullPlaylist,
} from "@/lib/music/app/get-playlist";
import { isLiked, likedCover } from "@/lib/music/app/liked";
import {
	isCustom,
	removeSongFromPlaylist,
	reorderPlaylist,
} from "@/lib/music/app/playlists";
import { move } from "@/lib/music/app/reorder";
import type { Playlist, Song } from "@/lib/music/model";
import { isPlaying } from "@/lib/music/views/stores/audio";
import { isQueue, playList } from "@/lib/music/views/stores/player";
import { expireRemoval, stageRemoval } from "@/lib/music/views/stores/removal";
import {
	clearSelection,
	selectAll,
	selectedSongs,
} from "@/lib/music/views/stores/selection";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import SelectionBar from "@/lib/music/views/ui/track/selection-bar";
import TrackActions from "@/lib/music/views/ui/track/track-actions";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { addError } from "@/lib/shared/views/stores/errors";
import { ConfirmModal } from "@/lib/shared/views/ui/components/confirm";
import { PageHeader } from "@/lib/shared/views/ui/components/page-header";

const storageKey = "playlist_filters";

type OrderType = "added" | "reverse" | "duration-asc" | "duration-desc";

type Filters = {
	order: OrderType;
};

/* ------------------ Filters ------------------ */

function isValidFilters(value: unknown): value is Filters {
	if (typeof value !== "object" || value === null) return false;

	const v = value as Record<string, unknown>;

	return (
		v.order === "added" ||
		v.order === "reverse" ||
		v.order === "duration-asc" ||
		v.order === "duration-desc"
	);
}

function loadFilters(): Filters {
	const fallback: Filters = {
		order: "added",
	};

	if (typeof window === "undefined") return fallback;

	try {
		const raw = localStorage.getItem(storageKey);
		if (!raw) return fallback;

		const parsed: unknown = JSON.parse(raw);
		return isValidFilters(parsed) ? parsed : fallback;
	} catch {
		return fallback;
	}
}

function saveFilters(data: Filters): void {
	localStorage.setItem(storageKey, JSON.stringify(data));
}

/* ------------------ Data helpers ------------------ */

function sortSongs(songs: Song[], order: OrderType): Song[] {
	switch (order) {
		case "reverse":
			return [...songs].reverse();

		case "duration-asc":
			return [...songs].sort((a, b) => a.duration - b.duration);

		case "duration-desc":
			return [...songs].sort((a, b) => b.duration - a.duration);

		default:
			return songs;
	}
}

function getInputValue(e: Event): string {
	return (e.target as HTMLInputElement).value;
}

function getSelectValue(e: Event): OrderType {
	return (e.target as HTMLSelectElement).value as OrderType;
}

/* ------------------ State ------------------ */

const initial = loadFilters();

const search = signal<string>("");
const order = signal<OrderType>(initial.order);

// persist reactively
effect(() => {
	saveFilters({
		order: order.value,
	});
});

/* ------------------ Component ------------------ */

export default function PlaylistPlain({ id }: { id: string }) {
	const refreshing = useSignal(false);
	const queryClient = useQueryClient();
	const custom = isCustom(id);

	useEffect(() => {
		search.value = "";
	}, [id]);

	const { data, isLoading, isError, refetch } = useQuery<Playlist>({
		queryKey: ["playlist", id],
		queryFn: () => getFullPlaylist(id),
	});

	const removeSong = useMutation({
		mutationFn: (ref: string) => removeSongFromPlaylist(id, ref),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["playlist", id] });
			queryClient.invalidateQueries({ queryKey: ["playlists"] });
		},
		onError: (error) => addError(error, "playlist"),
	});

	const removing = useSignal<Song[] | null>(null);

	const confirmRemove = () => {
		if (!removing.value) return;
		const order = (data?.songs ?? []).map((song) => song.id);
		for (const song of removing.value) removeSong.mutate(song.id);
		clearSelection();
		stageRemoval({
			playlistId: id,
			refs: removing.value.map((song) => song.id),
			order,
		});
		removing.value = null;
	};

	const reorder = useMutation({
		mutationFn: (refs: string[]) => reorderPlaylist(id, refs),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["playlist", id] });
			queryClient.invalidateQueries({ queryKey: ["playlists"] });
			expireRemoval(id);
		},
		onError: (error) => addError(error, "playlist"),
	});

	async function handleRefresh() {
		refreshing.value = true;
		await revalidateFullPlaylist(id).catch(() => {});
		await refetch();
		refreshing.value = false;
	}

	const liked = isLiked(id);
	const songs = data?.songs ?? [];
	const cover = liked
		? likedCover
		: (data?.cover ?? songs[0]?.album?.covers?.[0]);
	const covers = custom
		? [...new Set(songs.map((song) => song.album?.covers?.[0]))]
				.filter((value): value is string => !!value)
				.slice(0, 4)
		: [];
	const name = liked
		? t("music.likedSongs")
		: data?.name || t("music.playlist.defaultName");

	const query = search.value.trim().toLowerCase();
	const base =
		query.length === 0
			? songs
			: songs.filter((song) => {
					return (
						song.name.toLowerCase().includes(query) ||
						(song.artists ?? []).some((a) =>
							a.name.toLowerCase().includes(query),
						)
					);
				});
	const filteredSongs = sortSongs(base, order.value);
	const playable = filteredSongs.filter((song) => !song.broken);
	const canReorder =
		custom && query.length === 0 && order.value === "added" && songs.length > 1;

	const handleReorder = (from: number, to: number) => {
		const fromSong = songs[from];
		const toSong = songs[to];
		if (!fromSong || !toSong) return;
		reorder.mutate(
			move(
				songs.map((song) => song.id),
				fromSong.id,
				toSong.id,
			),
		);
	};

	if (isError) {
		return (
			<div className="flex min-h-0 flex-1 flex-col items-start gap-3 p-4 text-sm">
				<p className="text-red-400">{t("music.playlist.failed")}</p>
				<button
					type="button"
					onClick={handleRefresh}
					disabled={refreshing.value}
					className="rounded-lg border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 disabled:opacity-50 transition-colors cursor-pointer"
				>
					{t("common.retry")}
				</button>
			</div>
		);
	}

	return (
		<div className="flex min-h-0 flex-1 flex-col gap-4 pb-6">
			<PageHeader
				cover={cover}
				covers={covers}
				title={name}
				subtitle={t("music.trackCount", { count: songs.length })}
				link={
					data?.owner ? { to: `/${data.owner}`, label: data.owner } : undefined
				}
				linkReserve
				onPlay={playable.length > 0 ? () => void playList(playable) : undefined}
				playing={isQueue(playable) && isPlaying.value}
				actions={
					<TrackActions
						songs={songs}
						onSelectAll={selectAll}
						removable={custom}
						onRemove={
							custom
								? () => {
										removing.value =
											selectedSongs.value.length > 0
												? selectedSongs.value
												: songs;
									}
								: undefined
						}
					/>
				}
			/>

			{selectedSongs.value.length > 1 ? (
				<SelectionBar
					removable={custom}
					onRemove={
						custom
							? (next) => {
									removing.value = next;
								}
							: undefined
					}
				/>
			) : (
				<div
					className={cn(
						"grid gap-2",
						custom ? "grid-cols-[1fr_auto]" : "grid-cols-[1fr_auto_auto]",
					)}
				>
					{/* search */}
					<div className="relative flex-1">
						<Search
							size={16}
							className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500"
						/>
						<input
							type="text"
							placeholder={t("music.playlist.searchPlaceholder")}
							value={search.value}
							onInput={(e) => {
								search.value = getInputValue(e);
							}}
							className="h-10 w-full rounded-md border border-zinc-800 bg-zinc-950 pl-9 pr-9 text-sm text-white outline-none focus:border-zinc-600"
						/>
						{search.value && (
							<button
								type="button"
								onClick={() => {
									search.value = "";
								}}
								className="absolute right-2 top-1/2 -translate-y-1/2 flex h-5 w-5 cursor-pointer items-center justify-center rounded text-zinc-500 hover:text-white hover:bg-zinc-800"
								aria-label={t("music.playlist.clearSearch")}
							>
								<X size={14} />
							</button>
						)}
					</div>

					{/* order */}
					<div className="relative w-full">
						<ArrowUpDown
							size={16}
							className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500"
						/>
						<select
							value={order.value}
							onInput={(e) => {
								order.value = getSelectValue(e);
							}}
							aria-label={t("music.playlist.orderBy")}
							className="h-10 w-max appearance-none rounded-md border border-zinc-800 bg-zinc-950 pl-9 pr-3 text-sm text-white outline-none focus:border-zinc-600"
						>
							<option value="added">{t("music.playlist.order.added")}</option>
							<option value="reverse">
								{t("music.playlist.order.reverse")}
							</option>
							<option value="duration-asc">
								{t("music.playlist.order.durationAsc")}
							</option>
							<option value="duration-desc">
								{t("music.playlist.order.durationDesc")}
							</option>
						</select>
					</div>

					{/* refresh */}
					{!custom && (
						<button
							type="button"
							title={t("music.playlist.refresh")}
							aria-label={t("music.playlist.refresh")}
							onClick={handleRefresh}
							disabled={refreshing.value}
							className="flex h-10 w-10 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white disabled:opacity-50"
						>
							<RefreshCw
								size={16}
								class={refreshing.value ? "animate-spin" : ""}
							/>
						</button>
					)}
				</div>
			)}

			{isLoading ? (
				<div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
					<Loader size={40} class="animate-spin" />
				</div>
			) : (
				<Virtualization
					songs={filteredSongs}
					onRemove={
						custom
							? (next) => {
									removing.value = next;
								}
							: undefined
					}
					onReorder={canReorder ? handleReorder : undefined}
				/>
			)}

			<ConfirmModal
				open={!!removing.value}
				danger
				pending={removeSong.isPending}
				close={() => {
					removing.value = null;
				}}
				onConfirm={confirmRemove}
			/>
		</div>
	);
}
