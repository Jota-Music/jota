import { signal } from "@preact/signals";
import type { QueryClient } from "@tanstack/preact-query";
import { getFullPlaylist } from "@/lib/music/app/get-playlist";
import { addSongsToPlaylist, reorderPlaylist } from "@/lib/music/app/playlists";
import type { Playlist } from "@/lib/music/model";
import { addError } from "@/lib/shared/views/stores/errors";

export type Removal = {
	playlistId: string;
	refs: string[];
	order: string[];
};

export const removal = signal<Removal | null>(null);

export function stageRemoval(next: Removal) {
	removal.value = next;
}

export function dismissRemoval() {
	removal.value = null;
}

export function expireRemoval(id: string) {
	if (removal.value?.playlistId === id) dismissRemoval();
}

async function currentRefs(qc: QueryClient, id: string): Promise<string[]> {
	const queryKey = ["playlist", id];
	const data =
		qc.getQueryData<Playlist>(queryKey) ??
		(await qc.fetchQuery<Playlist>({
			queryKey,
			queryFn: () => getFullPlaylist(id),
		}));
	return (data?.songs ?? []).map((song) => song.id);
}

export async function undoRemoval(qc: QueryClient): Promise<void> {
	const current = removal.value;
	if (!current) return;
	dismissRemoval();
	try {
		await addSongsToPlaylist(current.playlistId, current.refs);
		const cur = await currentRefs(qc, current.playlistId);
		const order = new Set(current.order);
		const additions = cur.filter((ref) => !order.has(ref));
		await reorderPlaylist(current.playlistId, [...current.order, ...additions]);
		await qc.invalidateQueries({ queryKey: ["playlist", current.playlistId] });
		await qc.invalidateQueries({ queryKey: ["playlists"] });
	} catch (error) {
		addError(error, "playlist");
	}
}
