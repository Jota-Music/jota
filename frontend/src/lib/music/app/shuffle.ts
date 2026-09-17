// Deterministic Fisher-Yates, so the same seed yields the same order on every
// member of a room.

export function random(seed: number): () => number {
	let a = seed >>> 0;
	return () => {
		a = (a + 0x6d2b79f5) >>> 0;
		let t = a;
		t = Math.imul(t ^ (t >>> 15), t | 1);
		t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

export function permute<T>(items: T[], rand: () => number): T[] {
	const next = items.slice();
	for (let i = next.length - 1; i > 0; i--) {
		const j = Math.floor(rand() * (i + 1));
		[next[i], next[j]] = [next[j], next[i]];
	}
	return next;
}

export function seed(): number {
	return Math.floor(Math.random() * 0x100000000) >>> 0;
}
