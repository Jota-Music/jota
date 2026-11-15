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

// A drop takes the slot of the row it lands on, instead of landing beside it.
// The list the user drags may be the stored order or its reverse, so the
// positions are read on that view and written back to the order that is stored.
export function place(
	ids: string[],
	from: number,
	to: number,
	reversed = false,
): string[] {
	if (from === to || from < 0 || to < 0) return ids;
	const next = reversed ? [...ids].reverse() : [...ids];
	if (from >= next.length || to >= next.length) return ids;
	const [moved] = next.splice(from, 1);
	next.splice(to, 0, moved);
	return reversed ? next.reverse() : next;
}
