import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import { playerModalOpen } from "@/lib/music/views/stores/player";

export type EditorMode = "assign" | "search";

export const editingSong = signal<Song | null>(null);
export const editorMode = signal<EditorMode>("assign");

export function openEditor(song: Song) {
	editorMode.value = "assign";
	editingSong.value = song;
	// Re-opening an edit already on screen is a signal no-op: drop the
	// expanded player so the editor comes back to the front.
	playerModalOpen.value = false;
}
