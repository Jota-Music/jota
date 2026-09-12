import { signal } from "@preact/signals";
import type { Song } from "@/lib/music/model";

export const queue = signal<Song[]>([]);

export const currentIndex = signal<number>(-1);

export const playerSkeletonOn = signal(false);

export const showQueue = signal<boolean>(false);
