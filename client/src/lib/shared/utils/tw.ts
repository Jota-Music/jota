export function vs(to_get: string, options: Record<string, string>) {
	return options[to_get];
}

export function cn(...classes: Array<string | false | null | undefined>) {
	const to_join: string[] = [];

	for (const cls of classes) {
		if (typeof cls === "string") to_join.push(cls);
	}

	return to_join.join(" ");
}
