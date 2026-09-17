import { QueryClient, QueryClientProvider } from "@tanstack/preact-query";
import { lazy, Suspense } from "preact/compat";
import { useEffect } from "preact/hooks";
import { Route, Switch } from "wouter-preact";
import { syncSpotifyStatus } from "@/lib/auth/views/stores/session";
import { syncYouTubeStatus } from "@/lib/auth/views/stores/youtube";
import { RequireSpotify } from "@/lib/auth/views/ui/spotify-connect";
import { MainPage } from "@/lib/shared/views/ui/pages/main";
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
const SettingsPage = lazy(() => import("@/lib/shared/views/ui/pages/settings"));
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
		},
	},
});

function Router() {
	useEffect(() => {
		void syncSpotifyStatus();
		void syncYouTubeStatus();
		void loadVersion();
		void checkUpdate();
		void applyRelayOverride();
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
					<Route path="/:user">
						<RequireSpotify>
							<UserPage />
						</RequireSpotify>
					</Route>
					<Route path="/" component={MainPage} />
					<Route component={NotFoundPage} />
				</Switch>
			</Suspense>
		</QueryClientProvider>
	);
}

export default Router;
