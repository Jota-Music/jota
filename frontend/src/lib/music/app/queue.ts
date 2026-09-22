import type { Song } from "@/lib/music/model";

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
