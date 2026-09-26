import { signal } from "@preact/signals";
import { useMutation, useQuery, useQueryClient } from "@tanstack/preact-query";
import { Heart } from "lucide-preact";
import { isLiked } from "@/lib/music/app/liked";
import { isCustom } from "@/lib/music/app/playlists";
import {
	addSavedPlaylist,
	getSavedPlaylists,
	removeSavedPlaylist,
} from "@/lib/music/app/saved-playlist";
import { t } from "@/lib/shared/i18n";
import type { Action } from "@/lib/shared/views/ui/components/context-menu";

const key = ["saved-playlists"];

// Pending save intents, so a heart and a context menu entry for the same
// playlist never disagree while the store round-trips. An intent only covers
// that round-trip: it is dropped once the refetch lands, otherwise a stale one
// would shadow the store forever and a remount would still read it.
const overrides = signal<Record<string, boolean>>({});

function set(id: string, value: boolean) {
	overrides.value = { ...overrides.value, [id]: value };
}

function drop(id: string) {
	const { [id]: _dropped, ...rest } = overrides.value;
	overrides.value = rest;
}

export function saveLabel(saved: boolean): string {
	return saved ? t("music.unfollowPlaylist") : t("music.followPlaylist");
}

// A local playlist is already in the library, and Liked Songs is a context
// rather than a playlist, so neither is followable.
export function followable(id: string): boolean {
	return !isLiked(id) && !isCustom(id);
}

// useSaved keeps an external playlist in the library. It is source-agnostic:
// the id prefix picks the source, so a Spotify and a YouTube playlist are
// tracked the same way.
export function useSaved() {
	const queryClient = useQueryClient();

	const query = useQuery({ queryKey: key, queryFn: getSavedPlaylists });

	const saved = new Set((query.data ?? []).map((p) => p.id));
	const isSaved = (id: string) => overrides.value[id] ?? saved.has(id);

	// Awaiting the refetch is what makes dropping the intent safe: a failure
	// lands here too, so the heart falls back to whatever the store holds.
	const settle = async (id: string) => {
		await queryClient.invalidateQueries({ queryKey: key });
		drop(id);
	};

	const save = useMutation({
		mutationFn: (id: string) => addSavedPlaylist(id),
		onMutate: (id) => set(id, true),
		onSettled: (_data, _error, id) => settle(id),
	});

	const unsave = useMutation({
		mutationFn: (id: string) => removeSavedPlaylist(id),
		onMutate: (id) => set(id, false),
		onSettled: (_data, _error, id) => settle(id),
	});

	const follow = (id: string, next: boolean) => {
		if (next) {
			save.mutate(id);
		} else {
			unsave.mutate(id);
		}
	};

	const toggle = (id: string): Action => {
		const saved = isSaved(id);
		return {
			icon: Heart,
			label: saveLabel(saved),
			run: () => follow(id, !saved),
		};
	};

	return {
		saved: query.data ?? [],
		isLoading: query.isLoading,
		isSaved,
		follow,
		toggle,
		pending: save.isPending || unsave.isPending,
	};
}
