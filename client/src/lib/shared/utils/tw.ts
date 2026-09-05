export function vs(to_get: string, options: Record<string, string>) {
	return options[to_get];
}

export function cn(...classes: any[]) {
	const to_join = [];

	for (const cls of classes) {
		if (typeof cls === "string") to_join.push(cls);
	}

	return to_join.join(" ");
}
