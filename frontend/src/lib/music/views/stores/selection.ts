import { computed, signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";

export const selectedSongs = signal<Song[]>([]);
// Membership is asked once per visible row and once per action, so a linear
// scan over the selection made selecting everything in a big playlist
// quadratic. One set per selection change keeps every lookup flat.
export const selectedIds = computed(
	() => new Set(selectedSongs.value.map((s) => s.id)),
);
export const selectionActive = computed(() => selectedSongs.value.length > 0);
export const selectionAnchor = signal<number | null>(null);

export function setSelection(songs: Song[]) {
	selectedSongs.value = songs;
}

export function toggleSelection(song: Song) {
	const current = selectedSongs.value;
	const exists = selectedIds.value.has(song.id);
	selectedSongs.value = exists
		? current.filter((s) => s.id !== song.id)
		: [...current, song];
	if (selectedSongs.value.length === 0) selectionAnchor.value = null;
}

export function selectRange(songs: Song[], from: number, to: number) {
	const lo = Math.min(from, to);
	const hi = Math.max(from, to);
	setSelection(songs.slice(lo, hi + 1));
}

export function clearSelection() {
	selectedSongs.value = [];
	selectionAnchor.value = null;
}

export function selectAll(songs: Song[]) {
	const ids = selectedIds.value;
	const added = songs.filter((s) => !ids.has(s.id));
	if (added.length > 0)
		selectedSongs.value = [...selectedSongs.value, ...added];
}

// Songs queued for the "add to playlist" picker. A null value keeps it closed.
export const pickerSongs = signal<Song[] | null>(null);

export function openPicker(songs: Song[]) {
	if (songs.length === 0) return;
	pickerSongs.value = songs;
}

export function closePicker() {
	pickerSongs.value = null;
}
