import { QueryClient } from "@tanstack/preact-query";

export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			staleTime: 5 * 60 * 1000,
			// Long enough that coming back to a page paints the snapshot it had
			// instead of a spinner while the bridge answers again, short enough
			// that a long session does not keep every visited page in memory.
			gcTime: 30 * 60 * 1000,
			refetchOnWindowFocus: false,
		},
	},
});

// What each section of the sync document feeds. A section a pass did not apply
// leaves its queries alone, so a pass that only moved the order does not
// re-request the library. This map is the only list: a pass names sections, and
// the mapping from sections to query keys lives here, next to the client.
const sections = {
	playlists: ["playlists", "playlist"],
	follows: ["followed-users"],
	saved: ["saved-playlists"],
	order: ["shelf-order"],
	settings: ["sync-auto", "sync-import", "sync-interval"],
	sessions: [],
} satisfies Record<string, readonly string[]>;

export type Section = keyof typeof sections;

const all = Object.keys(sections) as Section[];

// Marks stale what the named sections feed, and nothing else: a search or a
// source's own lists are not in the document, so re-requesting them is waste.
export function invalidate(changed: readonly Section[] = all): void {
	const roots = new Set<string>();
	for (const section of changed) {
		for (const root of sections[section] ?? []) roots.add(root);
	}
	if (roots.size === 0) return;
	void queryClient.invalidateQueries({
		predicate: (query) => roots.has(String(query.queryKey[0])),
	});
}
