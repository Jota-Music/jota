import { QueryClient, QueryClientProvider } from "@tanstack/preact-query";
import { useEffect } from "preact/hooks";
import { Route, Switch } from "wouter-preact";
import { syncSpotifyStatus } from "@/lib/auth/views/stores/session";
import { RequireSpotify } from "@/lib/auth/views/ui/spotify-connect";
import { AlbumPage } from "@/lib/shared/views/ui/pages/album";
import { ArtistPage } from "@/lib/shared/views/ui/pages/artist";
import { MainPage } from "@/lib/shared/views/ui/pages/main";
import { NotFoundPage } from "@/lib/shared/views/ui/pages/not-found";
import { PlaylistPage } from "@/lib/shared/views/ui/pages/playlist";
import { SearchPage } from "@/lib/shared/views/ui/pages/search";
import SettingsPage from "@/lib/shared/views/ui/pages/settings";
import { UserPage } from "@/lib/shared/views/ui/pages/user";
import { YouTubeSearchPage } from "@/lib/shared/views/ui/pages/youtube-search";

const queryClient = new QueryClient();

function Router() {
	useEffect(() => {
		void syncSpotifyStatus();
	}, []);

	return (
		<QueryClientProvider client={queryClient}>
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
		</QueryClientProvider>
	);
}

export default Router;
