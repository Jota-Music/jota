import { beforeEach, expect, test } from "bun:test";
import { invalidate, queryClient } from "@/lib/shared/views/ui/query-client";

beforeEach(() => {
	queryClient.clear();
});

const stale = (key: unknown[]) =>
	queryClient.getQueryState(key)?.isInvalidated === true;

// The map is the only list: a pass names the sections it applied, and each one
// feeds its queries and nothing else. A key missing here never picks up a change.
test("a pass only marks what the sections it applied feed", () => {
	queryClient.setQueryData(["playlists"], []);
	queryClient.setQueryData(["playlist", "local:1"], {});
	queryClient.setQueryData(["followed-users"], []);
	queryClient.setQueryData(["saved-playlists"], []);
	queryClient.setQueryData(["shelf-order", "library", "me"], []);
	queryClient.setQueryData(["user-playlists", "youtube"], []);
	queryClient.setQueryData(["youtube-search", "q"], []);

	invalidate(["playlists", "follows"]);

	expect(stale(["playlists"])).toBe(true);
	expect(stale(["playlist", "local:1"])).toBe(true);
	expect(stale(["followed-users"])).toBe(true);

	// The sections the pass did not name keep their snapshot.
	expect(stale(["saved-playlists"])).toBe(false);
	expect(stale(["shelf-order", "library", "me"])).toBe(false);

	// A source's own lists and the search are not in the document.
	expect(stale(["user-playlists", "youtube"])).toBe(false);
	expect(stale(["youtube-search", "q"])).toBe(false);
});

// The idle shed and a session that changes ask for everything the document feeds.
test("everything the document feeds is marked together", () => {
	queryClient.setQueryData(["playlists"], []);
	queryClient.setQueryData(["saved-playlists"], []);
	queryClient.setQueryData(["shelf-order", "library", "me"], []);
	queryClient.setQueryData(["user-playlists", "youtube"], []);

	invalidate();

	expect(stale(["playlists"])).toBe(true);
	expect(stale(["saved-playlists"])).toBe(true);
	expect(stale(["shelf-order", "library", "me"])).toBe(true);
	expect(stale(["user-playlists", "youtube"])).toBe(false);
});

// The sessions section feeds no query: the source re-signs and announces it
// through its own event.
test("a session-only change marks nothing", () => {
	queryClient.setQueryData(["playlists"], []);

	invalidate(["sessions"]);

	expect(stale(["playlists"])).toBe(false);
});
