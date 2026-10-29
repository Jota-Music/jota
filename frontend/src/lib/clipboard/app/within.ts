// A provider that never settles must not hold its caller: wait at most ms and
// answer null instead. The late answer is dropped rather than kept for later.
export function within<T>(work: Promise<T>, ms: number): Promise<T | null> {
	let stop = () => {};
	const deadline = new Promise<null>((resolve) => {
		const timer = setTimeout(() => resolve(null), ms);
		stop = () => clearTimeout(timer);
	});
	return Promise.race([work.catch(() => null), deadline]).finally(() => stop());
}
