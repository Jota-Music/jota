import type { QueueAction, Song } from "@/lib/music/model";

// Shuffle permutes the queue, so a source list matches the queue by membership,
// not by order. Counts each id so duplicates are compared correctly.
export function sameQueue(a: Song[], b: Song[]): boolean {
	if (a.length !== b.length) return false;

	const count = new Map<string, number>();
	for (const song of a) count.set(song.id, (count.get(song.id) ?? 0) + 1);
	for (const song of b) {
		const n = count.get(song.id);
		if (!n) return false;
		count.set(song.id, n - 1);
	}
	return true;
}

export function isOwnList(
	source: string | null,
	queueSource: string | null,
	songs: Song[],
	queue: Song[],
): boolean {
	if (source != null) return source === queueSource;
	return sameQueue(queue, songs);
}

export function pinAfter(
	queue: Song[],
	index: number,
	from: number,
): { queue: Song[]; index: number } | null {
	const n = queue.length;
	if (from < 0 || from >= n) return null;
	if (index < 0 || index >= n) return null;
	if (from === index || from === index + 1) return null;

	const next = [...queue];
	const playingId = next[index].id;
	const [item] = next.splice(from, 1);

	const current = next.findIndex((song) => song.id === playingId);
	if (current === -1) return null;

	next.splice(Math.min(current + 1, next.length), 0, item);
	return { queue: next, index: current };
}

// Inserts songs right after the song with `afterId`, so the enqueuing device and
// every receiver land them on the same spot without shipping a whole snapshot.
// A null id means the head; an id that is gone means the end.
export function insertAfter(
	queue: Song[],
	songs: Song[],
	afterId: string | null,
): { queue: Song[]; index: number } {
	const at =
		afterId == null ? -1 : queue.findIndex((song) => song.id === afterId);
	const index = afterId == null ? 0 : at === -1 ? queue.length : at + 1;
	const next = [...queue];
	next.splice(index, 0, ...songs);
	return { queue: next, index };
}

// Reports the incremental edit that turns `prev` into `next`, when one exists: a
// contiguous insertion, or a resolved YouTube id on a single track. Anything else
// (a removal, a reorder, a whole new queue) returns null and is sent as a full
// snapshot instead.
export function delta(prev: Song[], next: Song[]): QueueAction | null {
	if (next.length > prev.length) {
		let start = 0;
		while (start < prev.length && prev[start].id === next[start].id) start++;
		const extra = next.length - prev.length;
		for (let i = start; i < prev.length; i++) {
			if (prev[i].id !== next[i + extra].id) return null;
		}
		return {
			action: "enqueue",
			songs: next.slice(start, start + extra),
			afterId: start > 0 ? next[start - 1].id : null,
		};
	}
	if (next.length !== prev.length) return null;

	let changed = -1;
	for (let i = 0; i < prev.length; i++) {
		if (prev[i].id !== next[i].id) return null;
		if (prev[i].youtubeId !== next[i].youtubeId) {
			if (changed !== -1) return null;
			changed = i;
		}
	}
	if (changed === -1) return null;
	return {
		action: "youtube",
		id: next[changed].id,
		youtubeId: next[changed].youtubeId ?? "",
	};
}
