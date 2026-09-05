import "@/lib/music/views/stores/audio";

import { signal } from "@preact/signals";
import { useQuery } from "@tanstack/preact-query";
import { Disc3, ListMusic, Loader } from "lucide-preact";
import { useEffect } from "preact/hooks";
import { Link, useParams } from "wouter-preact";
import { getArtist, getArtistDiscography } from "@/lib/music/app/get-artist";
import type { AlbumSummary, Song } from "@/lib/music/model";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

const view = signal<string>("tracks");

const groupOrder = ["álbum", "single", "compilación"] as const;

const groupLabels: Record<string, string> = {
	álbum: "Albums",
	single: "Singles",
	compilación: "Compilations",
	appears_on: "Appears on",
};

function groupLabel(key: string): string {
	return groupLabels[key] ?? key.charAt(0).toUpperCase() + key.slice(1);
}

function groupKeysOf(albums: AlbumSummary[] | undefined): string[] {
	if (!albums) return [];
	const keys: string[] = [];
	for (const key of groupOrder) {
		if (albums.some((a) => (a.group?.trim() || "álbum") === key)) {
			keys.push(key);
		}
	}
	for (const a of albums) {
		const key = a.group?.trim() || "álbum";
		if (!keys.includes(key)) {
			keys.push(key);
		}
	}
	return keys;
}

function groupCounts(
	albums: AlbumSummary[] | undefined,
): Record<string, number> {
	const counts: Record<string, number> = {};
	for (const a of albums ?? []) {
		const key = a.group?.trim() || "álbum";
		counts[key] = (counts[key] ?? 0) + 1;
	}
	return counts;
}

export function ArtistPage() {
	const { id } = useParams<{ id: string }>();
	useMeta(`Jota | Artist`, `Browse and listen to an artist on Jota`);

	const { data, isLoading, isError } = useQuery({
		queryKey: ["artist", id],
		queryFn: () => getArtist(id ?? ""),
		enabled: !!id,
	});

	const { data: disco, isLoading: discoLoading } = useQuery({
		queryKey: ["artist-discography", id],
		queryFn: () => getArtistDiscography(id ?? ""),
		enabled: !!id,
	});

	const tracks = (data?.tracks as Song[] | undefined) ?? [];

	useEffect(() => {
		view.value = "tracks";
	}, [id]);

	const counts = groupCounts(disco?.albums);
	const tabs = ["tracks", ...groupKeysOf(disco?.albums)];
	const activeTab = tabs.includes(view.value) ? view.value : "tracks";
	const selectedAlbums =
		activeTab === "tracks"
			? []
			: (disco?.albums ?? []).filter(
					(a) => (a.group?.trim() || "álbum") === activeTab,
				);

	if (isError) {
		return (
			<DefaultLayout class="gap-6 h-full">
				<div class="flex flex-col gap-6 h-full items-center justify-center">
					<p class="text-red-400">Failed to load artist</p>
				</div>
			</DefaultLayout>
		);
	}

	return (
		<DefaultLayout class="gap-4 h-full">
			<div class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
				<header class="flex items-center gap-4">
					{data?.imageUrl ? (
						<img
							src={data.imageUrl}
							alt=""
							class="h-16 w-16 shrink-0 rounded-md object-cover"
						/>
					) : (
						<div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-md bg-zinc-900 text-zinc-500">
							<Disc3 size={28} />
						</div>
					)}
					<div class="min-w-0">
						<h2 class="truncate text-xl font-semibold leading-tight">
							{data?.name ?? "Artista"}
						</h2>
					</div>
				</header>

				<div class="flex flex-wrap items-center gap-2">
					{tabs.map((tab) =>
						tab === "tracks" ? (
							<button
								key="tracks"
								type="button"
								onClick={() => {
									view.value = "tracks";
								}}
								class={`flex h-9 cursor-pointer items-center gap-2 rounded-md border px-3 text-sm transition-colors ${
									activeTab === "tracks"
										? "border-zinc-600 bg-zinc-900 text-white"
										: "border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white"
								}`}
							>
								<ListMusic size={14} />
								Top tracks
							</button>
						) : (
							<button
								key={tab}
								type="button"
								onClick={() => {
									view.value = tab;
								}}
								class={`flex h-9 cursor-pointer items-center gap-2 rounded-md border px-3 text-sm transition-colors ${
									activeTab === tab
										? "border-zinc-600 bg-zinc-900 text-white"
										: "border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white"
								}`}
							>
								<Disc3 size={14} />
								{groupLabel(tab)}
								{counts[tab] !== undefined && (
									<span class="text-xs text-zinc-500">({counts[tab]})</span>
								)}
							</button>
						),
					)}
					{discoLoading && (
						<Loader
							size={16}
							class="animate-spin text-zinc-500"
							aria-label="Loading discography"
						/>
					)}
				</div>

				{activeTab === "tracks" ? (
					isLoading ? (
						<div class="flex min-h-0 flex-1 items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
							<Loader size={24} class="animate-spin" />
						</div>
					) : tracks.length === 0 ? (
						<p class="text-sm text-zinc-500">No songs found.</p>
					) : (
						<Virtualization songs={tracks} />
					)
				) : (
					<div class="min-h-0 flex-1 overflow-y-auto rounded-md border border-zinc-800 bg-zinc-950">
						{discoLoading ? (
							<div class="flex items-center justify-center gap-2 p-8 text-sm text-zinc-400">
								<Loader size={24} class="animate-spin" />
							</div>
						) : selectedAlbums.length === 0 ? (
							<div class="flex min-h-32 items-center justify-center p-8 text-sm text-zinc-500">
								No {groupLabel(activeTab).toLowerCase()} found.
							</div>
						) : (
							<div class="grid grid-cols-2 gap-3 p-3 md:grid-cols-3">
								{selectedAlbums.map((a) => (
									<Link key={a.id} href={`/album/${a.id}`}>
										<div class="group flex flex-col gap-2 overflow-hidden rounded-md cursor-pointer">
											<div class="aspect-square w-full overflow-hidden rounded-md bg-zinc-900">
												{a.cover ? (
													<img
														src={a.cover}
														alt={a.name}
														loading="lazy"
														decoding="async"
														class="h-full w-full object-cover group-hover:opacity-80 transition-opacity"
													/>
												) : (
													<div class="flex h-full w-full items-center justify-center text-zinc-600">
														<Disc3 size={28} />
													</div>
												)}
											</div>
											<div class="flex flex-col">
												<h3 class="truncate text-sm font-medium text-zinc-300 group-hover:text-white transition-colors">
													{a.name}
												</h3>
												<p class="text-xs text-zinc-500">{a.year || "—"}</p>
											</div>
										</div>
									</Link>
								))}
							</div>
						)}
					</div>
				)}
			</div>
		</DefaultLayout>
	);
}

export default ArtistPage;
