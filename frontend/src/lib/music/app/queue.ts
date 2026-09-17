import type { QueueSong, Song } from "@/lib/music/model";

export function insertQueued(
	queue: QueueSong[],
	index: number,
	song: Song,
): { queue: QueueSong[]; index: number } {
	const next = [...queue];
	if (next.length === 0) {
		next.push(song);
		return { queue: next, index: 0 };
	}

	const at = index >= 0 && index < next.length ? index + 1 : next.length;
	next.splice(at, 0, { ...song, queued: true });
	return { queue: next, index };
}

export function pinAfter(
	queue: QueueSong[],
	index: number,
	from: number,
): { queue: QueueSong[]; index: number } | null {
	const n = queue.length;
	if (from < 0 || from >= n) return null;
	if (index < 0 || index >= n) return null;
	if (from === index || from === index + 1) return null;

	const next = [...queue];
	const playingId = next[index].id;
	const [item] = next.splice(from, 1);

	const current = next.findIndex((song) => song.id === playingId);
	if (current === -1) return null;

	next.splice(Math.min(current + 1, next.length), 0, {
		...item,
		queued: true,
	});
	return { queue: next, index: current };
}

export function clearQueued(queue: QueueSong[], index: number): QueueSong[] {
	const item = queue[index];
	if (!item?.queued) return queue;

	const next = queue.slice();
	next[index] = { ...item, queued: false };
	return next;
}
