import "@/lib/music/views/stores/audio";

import { signal } from "@preact/signals";
import { useQuery, useQueryClient } from "@tanstack/preact-query";
import { Disc3, ListMusic, Loader } from "lucide-preact";
import { useEffect } from "preact/hooks";
import { useParams } from "wouter-preact";
import { getArtist, getArtistDiscography } from "@/lib/music/app/get-artist";
import type { AlbumSummary, Song } from "@/lib/music/model";
import { playAlbum } from "@/lib/music/views/play";
import {
	isPlaying,
	isLoading as playbackLoading,
} from "@/lib/music/views/stores/audio";
import { playList } from "@/lib/music/views/stores/player";
import { queueSource } from "@/lib/music/views/stores/queue";
import { clearSelection, selectAll } from "@/lib/music/views/stores/selection";
import { Virtualization } from "@/lib/music/views/ui/playlist/virtualization";
import { type Item, Shelf } from "@/lib/music/views/ui/shelf";
import TrackActions from "@/lib/music/views/ui/track/track-actions";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { PageHeader } from "@/lib/shared/views/ui/components/page-header";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

const view = signal<string>("tracks");

const groupOrder = ["album", "single", "compilation"] as const;

const GROUP_LABELS: Record<string, string> = {
	album: "pages.artist.groups.album",
	single: "pages.artist.groups.single",
	compilation: "pages.artist.groups.compilation",
	appears_on: "pages.artist.groups.appears_on",
};

function groupLabel(key: string): string {
	const label = GROUP_LABELS[key];
	return label ? t(label) : key.charAt(0).toUpperCase() + key.slice(1);
}

function groupKeysOf(albums: AlbumSummary[] | undefined): string[] {
	if (!albums) return [];
	const keys: string[] = [];
	for (const key of groupOrder) {
		if (albums.some((a) => (a.group?.trim() || "album") === key)) {
			keys.push(key);
		}
	}
	for (const a of albums) {
		const key = a.group?.trim() || "album";
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
		const key = a.group?.trim() || "album";
		counts[key] = (counts[key] ?? 0) + 1;
	}
	return counts;
}

export function ArtistPage() {
	const queryClient = useQueryClient();
	const { id } = useParams<{ id: string }>();

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

	useEffect(() => {
		clearSelection();
	}, [activeTab]);
	const selectedAlbums =
		activeTab === "tracks"
			? []
			: (disco?.albums ?? []).filter(
					(a) => (a.group?.trim() || "album") === activeTab,
				);

	if (isError) {
		return (
			<DefaultLayout class="gap-6">
				<div class="flex flex-col gap-6 min-h-0 flex-1 items-center justify-center">
					<p class="text-red-400">{t("pages.artist.failed")}</p>
				</div>
			</DefaultLayout>
		);
	}

	return (
		<DefaultLayout class="gap-4">
			<div class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden pb-6">
				<PageHeader
					cover={data?.imageUrl}
					title={data?.name ?? t("pages.artist.defaultName")}
					onPlay={
						activeTab === "tracks" && tracks.length > 0
							? () => void playList(tracks, id)
							: undefined
					}
					playing={
						activeTab === "tracks" &&
						queueSource.value === id &&
						isPlaying.value
					}
					loading={
						activeTab === "tracks" &&
						queueSource.value === id &&
						playbackLoading.value
					}
					actions={
						activeTab === "tracks" ? (
							<TrackActions songs={tracks} onSelectAll={selectAll} />
						) : undefined
					}
				/>

				<div class="grid shrink-0 grid-cols-2 gap-2 md:grid-cols-4">
					{tabs.map((tab) =>
						tab === "tracks" ? (
							<button
								key="tracks"
								type="button"
								onClick={() => {
									view.value = "tracks";
								}}
								class={cn(
									"flex h-9 w-full cursor-pointer items-center justify-start gap-2 rounded-md border px-3 text-sm transition-colors",
									activeTab === "tracks"
										? "border-zinc-600 bg-zinc-900 text-white"
										: "border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white",
								)}
							>
								<ListMusic size={14} />
								{t("pages.artist.topTracks")}
							</button>
						) : (
							<button
								key={tab}
								type="button"
								onClick={() => {
									view.value = tab;
								}}
								class={cn(
									"flex h-9 w-full cursor-pointer items-center justify-start gap-2 rounded-md border px-3 text-sm transition-colors",
									activeTab === tab
										? "border-zinc-600 bg-zinc-900 text-white"
										: "border-zinc-800 bg-zinc-950 text-zinc-400 hover:text-white",
								)}
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
						<div class="flex items-center justify-start">
							<Loader
								size={16}
								class="animate-spin text-zinc-500"
								aria-label={t("pages.artist.loadingDiscography")}
							/>
						</div>
					)}
				</div>

				{activeTab === "tracks" ? (
					isLoading ? (
						<div class="flex min-h-0 flex-1 items-center justify-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 p-4 text-sm text-zinc-400">
							<Loader size={24} class="animate-spin" />
						</div>
					) : tracks.length === 0 ? (
						<p class="text-sm text-zinc-500">{t("pages.artist.noSongs")}</p>
					) : (
						<Virtualization songs={tracks} />
					)
				) : (
					<Shelf
						items={selectedAlbums.map(
							(a): Item => ({
								id: a.id,
								name: a.name,
								cover: a.cover,
								subtitle: a.year ? String(a.year) : undefined,
							}),
						)}
						to={(id) => `/album/${id}`}
						onPlay={(id) => void playAlbum(queryClient, id)}
						viewKey="artist_view"
						isLoading={discoLoading}
						emptyMessage={t("pages.artist.noGroup", {
							group: groupLabel(activeTab).toLowerCase(),
						})}
					/>
				)}
			</div>
		</DefaultLayout>
	);
}
