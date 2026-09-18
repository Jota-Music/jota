import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";

export const editingSong = signal<Song | null>(null);

export function openYoutubeEditor(song: Song): void {
	editingSong.value = song;
}

export function closeYoutubeEditor(): void {
	editingSong.value = null;
}
