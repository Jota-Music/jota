import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";

// Multi-select state for song rows. A double click/tap enters selection mode;
// while active, a single click toggles a row instead of playing it.
export const selectionActive = signal(false);
export const selectedSongs = signal<Song[]>([]);

export function startSelection(song: Song) {
	selectionActive.value = true;
	if (!selectedSongs.value.some((s) => s.id === song.id)) {
		selectedSongs.value = [...selectedSongs.value, song];
	}
}

export function toggleSelection(song: Song) {
	const current = selectedSongs.value;
	const exists = current.some((s) => s.id === song.id);
	selectedSongs.value = exists
		? current.filter((s) => s.id !== song.id)
		: [...current, song];
	if (selectedSongs.value.length === 0) selectionActive.value = false;
}

export function clearSelection() {
	selectionActive.value = false;
	selectedSongs.value = [];
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
