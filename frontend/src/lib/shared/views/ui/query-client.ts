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

// The roots a sync document pass can move: the playlists it stores, the friends
// followed on this install, the saved shelf and its order, and the two settings
// that travel. A pass cannot touch a search or a source's own lists, and marking
// them stale would re-request them for nothing.
const library = new Set([
	"playlists",
	"playlist",
	"saved-playlists",
	"followed-users",
	"shelf-order",
	"sync-auto",
	"sync-import",
	"sync-interval",
]);

export function invalidate(): void {
	void queryClient.invalidateQueries({
		predicate: (query) => library.has(String(query.queryKey[0])),
	});
}
