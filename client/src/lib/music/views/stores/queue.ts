import type { Song } from "@/lib/music/model";
import { getPlaybackSeconds, isPlaying } from "@/lib/music/views/stores/audio";
import { ws } from "@/lib/shared/api/socket";
import { signal } from "@preact/signals";

export const queue = signal<Song[]>([]);

export const currentIndex = signal<number>(-1);

export const playerSkeletonOn = signal(false);

export const showQueue = signal<boolean>(false);

export function shareSnapshot(o?: { position?: number; playing?: boolean }) {
    ws.send("snapshot", {
        queue: queue.value,
        index: currentIndex.value,
        playing: o?.playing !== undefined ? o.playing : isPlaying.value,
        position: o?.position !== undefined ? o.position : getPlaybackSeconds(),
    });
}

export function broadcastSeek(position: number) {
    ws.send("seek", { position });
}