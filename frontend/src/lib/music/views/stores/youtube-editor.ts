import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";

export type EditorMode = "assign" | "search";

export const editingSong = signal<Song | null>(null);
export const editorMode = signal<EditorMode>("assign");

export function openEditor(song: Song) {
	editorMode.value = "assign";
	editingSong.value = song;
}
