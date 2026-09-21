import { IdleReload, ReloadWindow } from "@bindings/app";
import { QueryClient, QueryClientProvider } from "@tanstack/preact-query";
import { lazy, Suspense } from "preact/compat";
import { useEffect } from "preact/hooks";
import { Route, Switch } from "wouter-preact";
import { syncSpotifyStatus } from "@/lib/auth/views/stores/session";
import { syncYouTubeStatus } from "@/lib/auth/views/stores/youtube";
import { RequireSpotify } from "@/lib/auth/views/ui/spotify-connect";
import { isPlaying } from "@/lib/music/views/stores/audio";
import { ContextMenu } from "@/lib/shared/views/ui/components/context-menu";
import { MainPage } from "@/lib/shared/views/ui/pages/main";
import SettingsPage from "@/lib/shared/views/ui/pages/settings";
import { applyRelayOverride } from "@/lib/sync/app/transport";
import { checkUpdate, loadVersion } from "@/lib/update/views/stores/update";

const PlaylistPage = lazy(() =>
	import("@/lib/shared/views/ui/pages/playlist").then((m) => ({
		default: m.PlaylistPage,
	})),
);
const ArtistPage = lazy(() =>
	import("@/lib/shared/views/ui/pages/artist").then((m) => ({
		default: m.ArtistPage,
	})),
);
const AlbumPage = lazy(() =>
	import("@/lib/shared/views/ui/pages/album").then((m) => ({
		default: m.AlbumPage,
	})),
);
const YouTubeSearchPage = lazy(() =>
	import("@/lib/shared/views/ui/pages/youtube-search").then((m) => ({
		default: m.YouTubeSearchPage,
	})),
);
const SearchPage = lazy(() =>
	import("@/lib/shared/views/ui/pages/search").then((m) => ({
		default: m.SearchPage,
	})),
);
const UserPage = lazy(() =>
	import("@/lib/shared/views/ui/pages/user").then((m) => ({
		default: m.UserPage,
	})),
);
const NotFoundPage = lazy(() =>
	import("@/lib/shared/views/ui/pages/not-found").then((m) => ({
		default: m.NotFoundPage,
	})),
);

const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			staleTime: 5 * 60 * 1000,
			gcTime: 60 * 1000,
			// A desktop window regains focus constantly; refetching every active
			// query on each focus just churns network and allocations.
			refetchOnWindowFocus: false,
		},
	},
});

// Hidden for this long means the app is genuinely idle, not just unfocused, so
// it is safe to drop every cached query and let them refetch on return.
const IDLE_SHED_MS = 5 * 60 * 1000;

// Opt-in: reload the webview when it has been hidden this long and playback is
// paused, which is the only way to flush WebKit's internal image caches.
const IDLE_RELOAD_MS = 10 * 60 * 1000;

function Router() {
	useEffect(() => {
		void syncSpotifyStatus();
		void syncYouTubeStatus();
		void loadVersion();
		void checkUpdate();
		void applyRelayOverride();
	}, []);

	useEffect(() => {
		let shed: ReturnType<typeof setTimeout> | undefined;
		let reload: ReturnType<typeof setTimeout> | undefined;
		const cancel = () => {
			if (shed !== undefined) clearTimeout(shed);
			if (reload !== undefined) clearTimeout(reload);
			shed = undefined;
			reload = undefined;
		};
		const onVisibility = () => {
			if (!document.hidden) {
				cancel();
				return;
			}
			shed = setTimeout(() => queryClient.clear(), IDLE_SHED_MS);
			reload = setTimeout(async () => {
				if (isPlaying.value) return;
				if (!(await IdleReload())) return;
				void ReloadWindow();
			}, IDLE_RELOAD_MS);
		};
		document.addEventListener("visibilitychange", onVisibility);
		return () => {
			document.removeEventListener("visibilitychange", onVisibility);
			cancel();
		};
	}, []);

	return (
		<QueryClientProvider client={queryClient}>
			<Suspense fallback={null}>
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
					<Route path="/search/youtube/:query" component={YouTubeSearchPage} />
					<Route path="/search/:type/:query">
						<RequireSpotify>
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
					<Route path="/" component={MainPage} />
					<Route component={NotFoundPage} />
				</Switch>
			</Suspense>
			<ContextMenu />
		</QueryClientProvider>
	);
}

export default Router;
