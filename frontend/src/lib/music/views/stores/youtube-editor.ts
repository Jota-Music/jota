import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";

export const editingSong = signal<Song | null>(null);
