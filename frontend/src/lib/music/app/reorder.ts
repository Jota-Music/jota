interface Identified {
	id: string;
}

// Sorts items by a saved id order, keeping items missing from the order at the
// end in their original relative position.
export function applyOrder<T extends Identified>(
	items: T[],
	ids: string[],
): T[] {
	if (ids.length === 0) return items;

	const rank = new Map(ids.map((id, index) => [id, index]));
	const last = ids.length;

	return [...items].sort(
		(a, b) => (rank.get(a.id) ?? last) - (rank.get(b.id) ?? last),
	);
}

export function move(ids: string[], fromId: string, toId: string): string[] {
	const from = ids.indexOf(fromId);
	const to = ids.indexOf(toId);
	if (from === -1 || to === -1 || from === to) return ids;

	const next = [...ids];
	const [moved] = next.splice(from, 1);
	next.splice(to, 0, moved);
	return next;
}
