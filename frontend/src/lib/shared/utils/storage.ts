export function get<T extends string | null = string>(
	key: string,
	fallback: T = "" as T,
): T {
	if (typeof window === "undefined") return fallback;
	try {
		return (localStorage.getItem(key) as T) ?? fallback;
	} catch {
		return fallback;
	}
}

export function set(key: string, value: string): void {
	if (typeof window === "undefined") return;
	try {
		localStorage.setItem(key, value);
	} catch {}
}
