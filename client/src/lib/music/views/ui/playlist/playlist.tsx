import { computed, effect, signal } from "@preact/signals";
import { useQuery } from "@tanstack/preact-query";
import { ArrowUpDown, Loader, Search } from "lucide-preact";
import { getFullPlaylist } from "@/lib/music/app/get-playlist";
import type { Song } from "@/lib/music/model";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";

const storageKey = "playlist_filters";

type OrderType = "added" | "reverse" | "duration-asc" | "duration-desc";

type Filters = {
	search: string;
	order: OrderType;
};

type PlaylistTrackItem = {
	track: Song;
};

type PlaylistResponse =
	| { songs: Song[] }
	| { items: PlaylistTrackItem[] }
	| { tracks: PlaylistTrackItem[] };

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

function normalizeSongs(data: PlaylistResponse | undefined): Song[] {
	if (!data) return [];

	if ("songs" in data && Array.isArray(data.songs)) {
		return data.songs;
	}

	if ("items" in data && Array.isArray(data.items)) {
		return data.items.map((item) => item.track);
	}

	if ("tracks" in data && Array.isArray(data.tracks)) {
		return data.tracks.map((item) => item.track);
	}

	return [];
}

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
	const { data, isLoading, isError } = useQuery<PlaylistResponse>({
		queryKey: ["playlist", id],
		queryFn: () => getFullPlaylist(id),
	});

	const songs = normalizeSongs(data);

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
			<div className="flex min-h-0 flex-1 flex-col p-4 text-sm text-red-400">
				Error loading playlist
			</div>
		);
	}

	return (
		<div className="flex min-h-0 flex-1 flex-col gap-4">
            {/* <pre>
                {JSON.stringify(data, null, 2)}
            </pre> */}
			<div className="grid grid-cols-[1fr_auto] gap-2">
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
						className="h-10 w-full rounded-md border border-zinc-800 bg-zinc-950 pl-9 pr-3 text-sm text-white outline-none focus:border-zinc-600"
					/>
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
