import {
	IdleReload,
	RamMB,
	ReloadWindow,
	RevalidateFollowing,
	RevalidateMyYouTubePlaylists,
} from "@bindings/app";
import { QueryClientProvider } from "@tanstack/preact-query";
import { Events } from "@wailsio/runtime";
import { lazy, Suspense } from "preact/compat";
import { useEffect } from "preact/hooks";
import { Route, Switch } from "wouter-preact";
import {
	spotifyUser,
	syncSpotifyStatus,
} from "@/lib/auth/views/stores/session";
import { syncYouTubeStatus } from "@/lib/auth/views/stores/youtube";
import { RequireSpotify } from "@/lib/auth/views/ui/spotify-connect";
import { autosave } from "@/lib/music/app/backup";
import { revalidateUserPlaylists } from "@/lib/music/app/get-user-playlists";
import { isPlaying } from "@/lib/music/views/stores/audio";
import { compactPlayer, Player } from "@/lib/music/views/ui/player";
import { cn } from "@/lib/shared/utils/tw";
import { ContextMenu } from "@/lib/shared/views/ui/components/context-menu";
import { AppShell } from "@/lib/shared/views/ui/layouts/app-shell";
import { MainPage } from "@/lib/shared/views/ui/pages/main";
import { queryClient } from "@/lib/shared/views/ui/query-client";
import { loadRelay } from "@/lib/sync/app/transport";
import { checkUpdate, loadVersion } from "@/lib/update/views/stores/update";

const pages = {
	playlist: () =>
		import("@/lib/shared/views/ui/pages/playlist").then((m) => ({
			default: m.PlaylistPage,
		})),
	artist: () =>
		import("@/lib/shared/views/ui/pages/artist").then((m) => ({
			default: m.ArtistPage,
		})),
	album: () =>
		import("@/lib/shared/views/ui/pages/album").then((m) => ({
			default: m.AlbumPage,
		})),
	radio: () =>
		import("@/lib/shared/views/ui/pages/radio").then((m) => ({
			default: m.RadioPage,
		})),
	youtubeSearch: () =>
		import("@/lib/shared/views/ui/pages/youtube-search").then((m) => ({
			default: m.YouTubeSearchPage,
		})),
	search: () =>
		import("@/lib/shared/views/ui/pages/search").then((m) => ({
			default: m.SearchPage,
		})),
	user: () =>
		import("@/lib/shared/views/ui/pages/user").then((m) => ({
			default: m.UserPage,
		})),
	notFound: () =>
		import("@/lib/shared/views/ui/pages/not-found").then((m) => ({
			default: m.NotFoundPage,
		})),
	// Settings is the biggest page nobody opens on launch, so it stays out of
	// the entry chunk; the idle warm below loads it before it is needed.
	settings: () =>
		import("@/lib/shared/views/ui/pages/settings").then((m) => ({
			default: m.default,
		})),
};

const PlaylistPage = lazy(pages.playlist);
const ArtistPage = lazy(pages.artist);
const AlbumPage = lazy(pages.album);
const RadioPage = lazy(pages.radio);
const YouTubeSearchPage = lazy(pages.youtubeSearch);
const SearchPage = lazy(pages.search);
const UserPage = lazy(pages.user);
const NotFoundPage = lazy(pages.notFound);
const SettingsPage = lazy(pages.settings);

// A store that opened late may have answered the first reads with an empty one:
// every local store reports nothing instead of failing while Badger opens, so the
// page asks again for everything it may have painted empty.
Events.On("store:ready", () => void queryClient.invalidateQueries());

// A background refresh pass revalidated cached catalog content; drop stale
// queries so open views pick up changes on their next render. Unmounted queries
// are not refetched, so untouched views keep their cached snapshot.
Events.On("refresh:updated", () => void queryClient.invalidateQueries());

// Cache entries never expire, so invalidating alone would just refetch the same
// stale bytes from the store. Revalidate over the live session first, then drop
// the queries so open views pick the fresh values up.
Events.On("youtube:session-updated", async () => {
	await syncYouTubeStatus();
	try {
		await RevalidateMyYouTubePlaylists();
	} catch {}
	await queryClient.invalidateQueries();
	// A cleared session is carried as a tombstone; write it now instead of
	// waiting for the next scheduled pass so it stops coming back.
	void autosave("auto").catch(() => {});
});

Events.On("spotify:reconnected", async () => {
	// The session is live again, so flip `connected` too. Clearing only
	// `degraded` would hide the banner while every gate stayed closed.
	await syncSpotifyStatus();
	const handle = spotifyUser.value;
	if (handle) {
		try {
			await revalidateUserPlaylists("spotify", handle);
			await RevalidateFollowing();
		} catch {}
	}
	await queryClient.invalidateQueries();
});

// Hidden this long means the app is genuinely idle, not just unfocused, so its
// cache is marked stale instead of being dropped: the views keep the data they
// already painted and refetch it behind the first frame on return. Discarding
// everything from memory is ReloadWindow's job below, which is deliberate.
const IDLE_SHED_MS = 5 * 60 * 1000;

// Reload the webview when it has been hidden this long and playback is paused,
// which is the only way to flush WebKit's internal image caches. This is on by
// default and can be disabled in settings. Once the hidden session crosses this
// RSS, the reload fires early instead of waiting out the full deadline; below
// it (or on platforms without /proc, where RamMB is 0) the deadline still
// applies.
const IDLE_RELOAD_MS = 10 * 60 * 1000;
const IDLE_RAM_MB = 600;
const RELOAD_POLL_MS = 30 * 1000;

function Router() {
	useEffect(() => {
		void syncSpotifyStatus();
		void syncYouTubeStatus();
		void loadVersion();
		void checkUpdate();
		void loadRelay();
	}, []);

	// Warm the route chunks once the first paint is done, so a navigation never
	// waits on the network and never falls into the Suspense boundary.
	useEffect(() => {
		const warm = () => {
			for (const load of Object.values(pages)) void load().catch(() => {});
		};
		if (typeof requestIdleCallback === "function") {
			const id = requestIdleCallback(warm, { timeout: 2000 });
			return () => cancelIdleCallback(id);
		}
		const id = setTimeout(warm, 200);
		return () => clearTimeout(id);
	}, []);

	useEffect(() => {
		let shed: ReturnType<typeof setTimeout> | undefined;
		let reload: ReturnType<typeof setInterval> | undefined;
		const cancel = () => {
			if (shed !== undefined) clearTimeout(shed);
			if (reload !== undefined) clearInterval(reload);
			shed = undefined;
			reload = undefined;
		};
		const onVisibility = () => {
			if (!document.hidden) {
				// A release being (re)created while the app is backgrounded can
				// leave the update snapshot stale; refresh on return.
				void checkUpdate();
				cancel();
				return;
			}
			shed = setTimeout(() => {
				void queryClient.invalidateQueries();
			}, IDLE_SHED_MS);
			const deadline = Date.now() + IDLE_RELOAD_MS;
			reload = setInterval(async () => {
				if (isPlaying.value) return;
				if (!(await IdleReload())) return;
				const mb = await RamMB();
				const due = Date.now() >= deadline;
				if (!due && mb > 0 && mb < IDLE_RAM_MB) return;
				void ReloadWindow();
			}, RELOAD_POLL_MS);
		};
		document.addEventListener("visibilitychange", onVisibility);
		return () => {
			document.removeEventListener("visibilitychange", onVisibility);
			cancel();
		};
	}, []);

	return (
		<QueryClientProvider client={queryClient}>
			<AppShell
				player={
					<div
						class={cn(
							"shrink-0 w-full max-w-2xl md:mx-auto relative",
							!compactPlayer.value && "md:mt-6 md:pb-6",
						)}
					>
						<Player />
					</div>
				}
			>
				<Suspense
					fallback={
						<div class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden pb-6" />
					}
				>
					<Switch>
						<Route path="/playlist/:id" component={PlaylistPage} />
						<Route path="/artist/:id">
							<RequireSpotify>
								<ArtistPage />
							</RequireSpotify>
						</Route>
						<Route path="/album/:id">
							<RequireSpotify>
								<AlbumPage />
							</RequireSpotify>
						</Route>
						<Route path="/radio/:id">
							<RequireSpotify live>
								<RadioPage />
							</RequireSpotify>
						</Route>
						<Route
							path="/search/youtube/:query"
							component={YouTubeSearchPage}
						/>
						<Route path="/search/:type/:query">
							<RequireSpotify live>
								<SearchPage />
							</RequireSpotify>
						</Route>
						<Route path="/settings" component={SettingsPage} />
						<Route path="/spotify/user/:username">
							<RequireSpotify>
								<UserPage />
							</RequireSpotify>
						</Route>
						<Route path="/youtube/user/:channelId" component={UserPage} />
						<Route path="/playlists" component={MainPage} />
						<Route path="/following" component={MainPage} />
						<Route path="/rooms" component={MainPage} />
						<Route path="/" component={MainPage} />
						<Route component={NotFoundPage} />
					</Switch>
				</Suspense>
			</AppShell>
			<ContextMenu />
		</QueryClientProvider>
	);
}

export default Router;
