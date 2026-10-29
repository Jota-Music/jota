import { expect, test } from "bun:test";
import { invalidate, queryClient } from "@/lib/shared/views/ui/query-client";

// The key list is the fragile half of the narrow invalidation: a query the sync
// document feeds that is missing here would never pick up a change.
test("a sync pass only marks what the document feeds", () => {
	queryClient.setQueryData(["playlists"], []);
	queryClient.setQueryData(["playlist", "local:1"], {});
	queryClient.setQueryData(["saved-playlists"], []);
	queryClient.setQueryData(["followed-users"], []);
	queryClient.setQueryData(["shelf-order", "library", "me"], []);
	queryClient.setQueryData(["following", "me"], []);
	queryClient.setQueryData(["friends", "me"], []);
	queryClient.setQueryData(["user-playlists", "youtube"], []);
	queryClient.setQueryData(["youtube-search", "q"], []);

	invalidate();

	const stale = (key: unknown[]) =>
		queryClient.getQueryState(key)?.isInvalidated === true;

	expect(stale(["playlists"])).toBe(true);
	expect(stale(["playlist", "local:1"])).toBe(true);
	expect(stale(["saved-playlists"])).toBe(true);
	expect(stale(["followed-users"])).toBe(true);
	expect(stale(["shelf-order", "library", "me"])).toBe(true);

	// A source's own lists and the search are not in the document.
	expect(stale(["following", "me"])).toBe(false);
	expect(stale(["friends", "me"])).toBe(false);
	expect(stale(["user-playlists", "youtube"])).toBe(false);
	expect(stale(["youtube-search", "q"])).toBe(false);
});
