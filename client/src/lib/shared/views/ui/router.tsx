import { QueryClient, QueryClientProvider } from "@tanstack/preact-query";
import { Route, Switch } from "wouter-preact";
import { AlbumPage } from "@/lib/shared/views/ui/pages/album";
import { ArtistPage } from "@/lib/shared/views/ui/pages/artist";
import { MainPage } from "@/lib/shared/views/ui/pages/main";
import { PlaylistPage } from "@/lib/shared/views/ui/pages/playlist";
import RegisterPage from "@/lib/shared/views/ui/pages/register";
import SettingsPage from "@/lib/shared/views/ui/pages/settings";
import { UserPage } from "@/lib/shared/views/ui/pages/user";
import { SearchPage } from "@/lib/shared/views/ui/pages/search";

const queryClient = new QueryClient();

function Router() {
	return (
		<QueryClientProvider client={queryClient}>
			<Switch>
				<Route path="/playlist/:id" component={PlaylistPage} />
				<Route path="/artist/:id" component={ArtistPage} />
				<Route path="/album/:id" component={AlbumPage} />
				<Route path="/register" component={RegisterPage} />
				<Route path="/settings" component={SettingsPage} />
				<Route path="/join/:room" component={MainPage} />
				<Route path="/search/:type/:query" component={SearchPage} />
				<Route path="/:user" component={UserPage} />
				<Route path="/" component={MainPage} />
				<Route>404: No such page!</Route>
			</Switch>
		</QueryClientProvider>
	);
}

export default Router;
