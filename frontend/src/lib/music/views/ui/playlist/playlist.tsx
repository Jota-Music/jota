import { effect, signal, useSignal } from "@preact/signals";
import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import {
	ArrowUpDown,
	Check,
	EllipsisVertical,
	ListChecks,
	ListPlus,
	Pencil,
	RefreshCw,
	Search,
	Square,
	SquarePlus,
	Trash2,
	X,
} from "lucide-preact";
import { useEffect, useMemo, useRef } from "preact/hooks";
import { useLocation } from "wouter-preact";
import { spotifyConnected } from "@/lib/auth/views/stores/session";
import {
	getFullPlaylist,
	revalidateFullPlaylist,
} from "@/lib/music/app/get-playlist";
import { isLiked, likedCover } from "@/lib/music/app/liked";
import { playlistSource } from "@/lib/music/app/playlist-source";
import {
	deletePlaylist,
	isCustom,
	removeSongsFromPlaylist,
	renamePlaylist,
	reorderPlaylist,
} from "@/lib/music/app/playlists";
import { pageLink } from "@/lib/music/app/share";
import type { Playlist, Song } from "@/lib/music/model";
import {
	isPlaying,
	isLoading as playbackLoading,
} from "@/lib/music/views/stores/audio";
import { enqueueMany, playList } from "@/lib/music/views/stores/player";
import { queueSource } from "@/lib/music/views/stores/queue";
import { expireRemoval, stageRemoval } from "@/lib/music/views/stores/removal";
import {
	clearSelection,
	openPicker,
	selectAll,
	selectedIds,
	selectedSongs,
} from "@/lib/music/views/stores/selection";
import { SaveButton } from "@/lib/music/views/ui/follow";
import { followable, useSaved } from "@/lib/music/views/ui/playlist/saved";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { shareLink } from "@/lib/music/views/ui/share";
import { Rows } from "@/lib/music/views/ui/track/rows";
import SelectionBar from "@/lib/music/views/ui/track/selection-bar";
import { move } from "@/lib/shared/app/reorder";
import { t } from "@/lib/shared/i18n";
import { translateError } from "@/lib/shared/i18n/errors";
import { cn } from "@/lib/shared/utils/tw";
import { addError } from "@/lib/shared/views/stores/errors";
import { ConfirmModal } from "@/lib/shared/views/ui/components/confirm";
import {
	type Action,
	openContextMenu,
} from "@/lib/shared/views/ui/components/context-menu";
import { Modal } from "@/lib/shared/views/ui/components/modal";
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
	const deletePlaylistConfirm = useSignal(false);
	const renamePlaylistOpen = useSignal(false);
	const renameDraft = useSignal("");
	const renameInputRef = useRef<HTMLInputElement>(null);
	const queryClient = useQueryClient();
	const [, navigate] = useLocation();
	const custom = isCustom(id);

	useEffect(() => {
		search.value = "";
	}, [id]);

	const { data, isLoading, isError, error, refetch } = useQuery<Playlist>({
		queryKey: ["playlist", id],
		queryFn: () => getFullPlaylist(id),
	});

	const removeSongs = useMutation({
		mutationFn: (refs: string[]) => removeSongsFromPlaylist(id, refs),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["playlist", id] });
			queryClient.invalidateQueries({ queryKey: ["playlists"] });
		},
		onError: (error) => addError(error, "playlist"),
	});

	const removing = useSignal<Song[] | null>(null);

	const confirmRemove = () => {
		if (!removing.value) return;
		const refs = removing.value.map((song) => song.id);
		const order = (data?.songs ?? []).map((song) => song.id);
		removeSongs.mutate(refs);
		clearSelection();
		stageRemoval({ playlistId: id, refs, order });
		removing.value = null;
	};

	const confirmDeletePlaylist = async () => {
		try {
			await deletePlaylist(id);
			void queryClient.invalidateQueries({ queryKey: ["playlists"] });
			clearSelection();
			deletePlaylistConfirm.value = false;
			navigate("/");
		} catch (error) {
			addError(error, "playlist");
		}
	};

	const openRenamePlaylist = () => {
		renameDraft.value = name;
		renamePlaylistOpen.value = true;
	};

	useEffect(() => {
		if (!renamePlaylistOpen.value) return;
		const id = requestAnimationFrame(() => {
			const input = renameInputRef.current;
			if (!input) return;
			input.focus();
		});
		return () => cancelAnimationFrame(id);
	}, [renamePlaylistOpen.value]);

	const confirmRenamePlaylist = async () => {
		const trimmed = renameDraft.value.trim();
		if (!trimmed) return;
		try {
			await renamePlaylist(id, trimmed);
			await queryClient.invalidateQueries({ queryKey: ["playlist", id] });
			void queryClient.invalidateQueries({ queryKey: ["playlists"] });
			renamePlaylistOpen.value = false;
		} catch (error) {
			addError(error, "playlist");
		}
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
	const name = liked
		? t("music.likedSongs")
		: data?.name || t("music.playlist.defaultName");

	const query = search.value.trim().toLowerCase();
	const { covers, filteredSongs, playable, canReorder } = useMemo(() => {
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
		const sorted = sortSongs(base, order.value);
		return {
			covers: custom
				? [...new Set(songs.map((song) => song.album?.covers?.[0]))]
						.filter((value): value is string => !!value)
						.slice(0, 4)
				: [],
			filteredSongs: sorted,
			playable: sorted.filter((song) => !song.broken),
			canReorder:
				custom &&
				query.length === 0 &&
				order.value === "added" &&
				songs.length > 1,
		};
	}, [songs, custom, query, order.value]);

	const hasBrokenSpotify = songs.some(
		(song) => song.broken && song.id.startsWith("spotify:"),
	);

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

	const ownerLink = (() => {
		if (!data?.owner) return undefined;
		if (playlistSource(id) === "youtube") {
			return data.ownerId
				? { to: `/youtube/user/${data.ownerId}`, label: data.owner }
				: undefined;
		}
		return { to: `/spotify/user/${data.owner}`, label: data.owner };
	})();

	if (isError) {
		return (
			<div className="flex min-h-0 flex-1 flex-col items-start gap-3 p-4 text-sm">
				<p className="text-red-400">{t("music.playlist.failed")}</p>
				<p className="max-w-xl text-xs break-words text-zinc-500">
					{translateError(error)}
				</p>
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
			<PlaylistHeader
				id={id}
				cover={cover}
				covers={covers}
				name={name}
				trackCount={songs.length}
				playable={playable}
				link={ownerLink}
				custom={custom}
				songs={songs}
				reserve={isLoading}
				onDeletePlaylist={() => (deletePlaylistConfirm.value = true)}
				onRenamePlaylist={custom ? openRenamePlaylist : undefined}
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

			{!isLoading && hasBrokenSpotify && !spotifyConnected.value && (
				<div class="rounded-md border border-amber-900/60 bg-amber-950/40 px-3 py-2 text-xs text-amber-200">
					{t("music.playlist.spotifyLoginNeeded")}
				</div>
			)}

			{isLoading ? (
				<Rows />
			) : (
				<Virtualization
					songs={filteredSongs}
					sourceId={id}
					resetKey={`${order.value}|${query}`}
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
				open={deletePlaylistConfirm.value}
				danger
				close={() => (deletePlaylistConfirm.value = false)}
				onConfirm={() => void confirmDeletePlaylist()}
			/>
			<ConfirmModal
				open={!!removing.value}
				danger
				pending={removeSongs.isPending}
				close={() => {
					removing.value = null;
				}}
				onConfirm={confirmRemove}
			/>

			<Modal
				open={renamePlaylistOpen.value}
				close={() => {
					renamePlaylistOpen.value = false;
				}}
				labelledBy="rename-playlist-title"
			>
				<form
					class="flex flex-col gap-4 px-6 pt-6 pb-6"
					onSubmit={(e) => {
						e.preventDefault();
						void confirmRenamePlaylist();
					}}
				>
					<h2
						id="rename-playlist-title"
						class="text-lg font-bold text-zinc-100"
					>
						{t("music.custom.rename")}
					</h2>
					<label for="rename-playlist-input" class="sr-only">
						{t("music.custom.rename")}
					</label>
					<input
						ref={renameInputRef}
						id="rename-playlist-input"
						type="text"
						value={renameDraft.value}
						onInput={(e) => {
							renameDraft.value = (e.target as HTMLInputElement).value;
						}}
						onKeyDown={(e) => {
							if (e.key === "Escape") {
								renamePlaylistOpen.value = false;
							}
						}}
						class="h-10 w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 text-sm text-white outline-none focus:border-zinc-600"
					/>
					<div class="flex justify-end gap-2">
						<button
							type="button"
							onClick={() => {
								renamePlaylistOpen.value = false;
							}}
							title={t("common.cancel")}
							aria-label={t("common.cancel")}
							class="flex size-10 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:text-white"
						>
							<X size={18} />
						</button>
						<button
							type="submit"
							title={t("common.save")}
							aria-label={t("common.save")}
							class="flex size-10 items-center justify-center rounded-lg bg-(--dominant-color) text-(--binary-color) transition-opacity hover:opacity-75"
						>
							<Check size={18} />
						</button>
					</div>
				</form>
			</Modal>
		</div>
	);
}

// Playback state is read here so a play/pause toggle re-renders the header
// alone, instead of re-deriving the whole list in the page body.
function PlaylistHeader({
	id,
	cover,
	covers,
	name,
	trackCount,
	playable,
	link,
	custom,
	songs,
	reserve,
	onDeletePlaylist,
	onRenamePlaylist,
}: {
	id: string;
	cover?: string;
	covers: string[];
	name: string;
	trackCount: number;
	playable: Song[];
	link?: { to: string; label: string };
	custom: boolean;
	songs: Song[];
	reserve?: boolean;
	onDeletePlaylist: () => void;
	onRenamePlaylist?: () => void;
}) {
	// queueSource instead of isQueue: membership walks the whole queue, so the
	// header re-rendered and re-filtered the track list on every queue edit.
	const active = queueSource.value === id;
	const { toggle } = useSaved();
	const follow = followable(id);
	const menuActions: Action[] = [];
	if (follow) menuActions.push(toggle(id));
	menuActions.push(...shareLink(pageLink("playlist", id)));
	const tracks = songs.filter((song) => !song.broken);
	if (tracks.length > 0) {
		menuActions.push(
			{
				icon: ListPlus,
				label: t("music.track.addQueue"),
				run: () => enqueueMany(tracks),
			},
			{
				icon: SquarePlus,
				label: t("music.custom.addTo"),
				run: () => openPicker(tracks),
			},
		);
		const allSelected = tracks.every((song) => selectedIds.value.has(song.id));
		menuActions.push({
			icon: allSelected ? Square : ListChecks,
			label: allSelected
				? t("music.custom.deselectAll")
				: t("music.custom.selectAll"),
			run: () => {
				if (allSelected) clearSelection();
				else selectAll(tracks);
			},
		});
	}
	if (custom && onRenamePlaylist) {
		menuActions.push({
			icon: Pencil,
			label: t("music.custom.rename"),
			run: onRenamePlaylist,
		});
	}
	if (custom) {
		menuActions.push({
			icon: Trash2,
			label: t("music.custom.deletePlaylist"),
			danger: true,
			run: onDeletePlaylist,
		});
	}
	return (
		<PageHeader
			cover={cover}
			covers={covers}
			title={name}
			subtitle={t("music.trackCount", { count: trackCount })}
			link={link}
			linkReserve
			beforePlay={
				follow || custom
					? follow && <SaveButton id={id} variant="toolbar" />
					: undefined
			}
			onPlay={
				playable.length > 0 ? () => void playList(playable, id) : undefined
			}
			playing={active && isPlaying.value}
			loading={active && playbackLoading.value}
			actions={<PlaylistActions actions={menuActions} reserve={reserve} />}
		/>
	);
}

function PlaylistActions({
	actions,
	reserve,
}: {
	actions: Action[];
	reserve?: boolean;
}) {
	if (actions.length === 0 && !reserve) return null;
	return (
		<button
			type="button"
			title={t("music.track.actions")}
			aria-label={t("music.track.actions")}
			aria-haspopup="menu"
			disabled={actions.length === 0}
			onPointerDown={(event) => {
				event.preventDefault();
				event.stopPropagation();
			}}
			onClick={(event) => {
				event.stopPropagation();
				openContextMenu(actions, event);
			}}
			class="shrink-0 cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800/80 hover:text-white disabled:cursor-default disabled:opacity-40"
		>
			<EllipsisVertical size={18} />
		</button>
	);
}
