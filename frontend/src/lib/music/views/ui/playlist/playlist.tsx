import { computed, effect, signal } from "@preact/signals";
import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { ArrowUpDown, Loader, RefreshCw, Search, X } from "lucide-preact";
import { getFullPlaylist } from "@/lib/music/app/get-playlist";
import type { Playlist, Song } from "@/lib/music/model";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { PageHeader } from "@/lib/shared/views/ui/components/page-header";

const storageKey = "playlist_filters";

type OrderType = "added" | "reverse" | "duration-asc" | "duration-desc";

type Filters = {
	search: string;
	order: OrderType;
};

/* ------------------ Filters ------------------ */

function isValidFilters(value: unknown): value is Filters {
	if (typeof value !== "object" || value === null) return false;

	const v = value as Record<string, unknown>;

	return (
		typeof v.search === "string" &&
		(v.order === "added" ||
			v.order === "reverse" ||
			v.order === "duration-asc" ||
			v.order === "duration-desc")
	);
}

function loadFilters(): Filters {
	const fallback: Filters = {
		search: "",
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
			return [...songs].sort((a, b) => (a.duration ?? 0) - (b.duration ?? 0));

		case "duration-desc":
			return [...songs].sort((a, b) => (b.duration ?? 0) - (a.duration ?? 0));

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

const search = signal<string>(initial.search);
const order = signal<OrderType>(initial.order);

// persist reactively
effect(() => {
	saveFilters({
		search: search.value,
		order: order.value,
	});
});

/* ------------------ Component ------------------ */

export default function PlaylistPlain({ id }: { id: string }) {
	const queryClient = useQueryClient();
	const refreshing = signal(false);

	const { data, isLoading, isError } = useQuery<Playlist>({
		queryKey: ["playlist", id],
		queryFn: () => getFullPlaylist(id),
	});

	async function handleRefresh() {
		refreshing.value = true;
		queryClient.removeQueries({ queryKey: ["playlist", id] });
		await queryClient.fetchQuery({
			queryKey: ["playlist", id],
			queryFn: () => getFullPlaylist(id),
		});
		refreshing.value = false;
	}

	const songs = data?.songs ?? [];
	const cover = data?.cover ?? songs[0]?.album?.covers?.[0];
	const name = data?.name || "Playlist";

	const filteredSongs = computed(() => {
		const query = search.value.trim().toLowerCase();

		const base =
			query.length === 0
				? songs
				: songs.filter((song) => {
						return (
							song.name.toLowerCase().includes(query) ||
							song.artists.some((a) => a.name.toLowerCase().includes(query))
						);
					});

		return sortSongs(base, order.value);
	});

	if (isError) {
		return (
			<div className="flex min-h-0 flex-1 flex-col items-start gap-3 p-4 text-sm">
				<p className="text-red-400">Failed to load playlist</p>
				<button
					type="button"
					onClick={handleRefresh}
					disabled={refreshing.value}
					className="rounded-lg border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800 disabled:opacity-50 transition-colors cursor-pointer"
				>
					Retry
				</button>
			</div>
		);
	}

	return (
		<div className="flex min-h-0 flex-1 flex-col gap-4 pb-6">
			<PageHeader
				cover={cover}
				title={name}
				subtitle={`${songs.length} track${songs.length === 1 ? "" : "s"}`}
			/>

			<div className="grid grid-cols-[1fr_auto_auto] gap-2">
				{/* search */}
				<div className="relative flex-1">
					<Search
						size={16}
						className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500"
					/>
					<input
						type="text"
						placeholder="Search songs or artists..."
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
							aria-label="Clear search"
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
						aria-label="Order by"
						className="h-10 w-max appearance-none rounded-md border border-zinc-800 bg-zinc-950 pl-9 pr-3 text-sm text-white outline-none focus:border-zinc-600"
					>
						<option value="added">Oldest</option>
						<option value="reverse">Newest</option>
						<option value="duration-asc">Shortest</option>
						<option value="duration-desc">Longest</option>
					</select>
				</div>

				{/* refresh */}
				<button
					type="button"
					onClick={handleRefresh}
					disabled={refreshing.value}
					className="flex h-10 w-10 cursor-pointer items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white disabled:opacity-50"
				>
					<RefreshCw size={16} class={refreshing.value ? "animate-spin" : ""} />
				</button>
			</div>

			{isLoading ? (
				<div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
					<Loader size={40} class="animate-spin" />
				</div>
			) : (
				<Virtualization songs={filteredSongs.value} />
			)}
		</div>
	);
}
