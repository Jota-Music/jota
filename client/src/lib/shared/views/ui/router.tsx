import { effect } from "@preact/signals";
import { QueryClient, QueryClientProvider } from "@tanstack/preact-query";
import type { ComponentChildren } from "preact";
import { useEffect, useState } from "preact/hooks";
import { Redirect, Route, Switch } from "wouter-preact";
import {
	loginSpotifyWithCode,
	spotifyConnected,
	syncSpotifyStatus,
} from "@/lib/auth/views/stores/session";
import { AlbumPage } from "@/lib/shared/views/ui/pages/album";
import { ArtistPage } from "@/lib/shared/views/ui/pages/artist";
import LoginPage from "@/lib/shared/views/ui/pages/login";
import { MainPage } from "@/lib/shared/views/ui/pages/main";
import { PlaylistPage } from "@/lib/shared/views/ui/pages/playlist";
import { SearchPage } from "@/lib/shared/views/ui/pages/search";
import { UserPage } from "@/lib/shared/views/ui/pages/user";

const queryClient = new QueryClient();

function SpotifyGate({ children }: { children: ComponentChildren }) {
	const [known, setKnown] = useState(false);
	const [connected, setConnected] = useState(spotifyConnected.value);

	useEffect(() => {
		void syncSpotifyStatus().finally(() => setKnown(true));
		const unsubscribe = effect(() => {
			setConnected(spotifyConnected.value);
		});
		return () => unsubscribe();
	}, []);

	useEffect(() => {
		if (connected) return;
		const timer = window.setInterval(() => {
			void syncSpotifyStatus().catch(() => {});
		}, 2500);
		return () => window.clearInterval(timer);
	}, [connected]);

	if (!known) {
		return (
			<div class="h-dvh flex items-center justify-center bg-stone-950">
				<span class="text-zinc-500">…</span>
			</div>
		);
	}

	if (!connected) return <LoginPage />;

	return children;
}

function OAuthCallback() {
	useEffect(() => {
		if (window.location.pathname !== "/login") return;
		const params = new URLSearchParams(window.location.search);
		const code = params.get("code");
		const done = () => window.location.replace("/");
		if (!code) {
			done();
			return;
		}
		void loginSpotifyWithCode(code).finally(done);
	}, []);
	return null;
}

function Router() {
	return (
		<QueryClientProvider client={queryClient}>
			<OAuthCallback />
			<SpotifyGate>
				<Switch>
					<Route path="/playlist/:id" component={PlaylistPage} />
					<Route path="/artist/:id" component={ArtistPage} />
					<Route path="/album/:id" component={AlbumPage} />
					{/* TODO: re-enable settings route once the feature is needed again */}
					{/* <Route path="/settings" component={SettingsPage} /> */}
					<Route path="/join/:room" component={MainPage} />
					<Route path="/search/:type/:query" component={SearchPage} />
					<Route path="/login">
						<Redirect to="/" />
					</Route>
					<Route path="/:user" component={UserPage} />
					<Route path="/" component={MainPage} />
					<Route>404: No such page!</Route>
				</Switch>
			</SpotifyGate>
		</QueryClientProvider>
	);
}

export default Router;
