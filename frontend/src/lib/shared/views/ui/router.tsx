import { effect } from "@preact/signals";
import { QueryClient, QueryClientProvider } from "@tanstack/preact-query";
import type { ComponentChildren } from "preact";
import { useEffect, useState } from "preact/hooks";
import { Route, Switch } from "wouter-preact";
import {
	spotifyConnected,
	syncSpotifyStatus,
} from "@/lib/auth/views/stores/session";
import { AlbumPage } from "@/lib/shared/views/ui/pages/album";
import { ArtistPage } from "@/lib/shared/views/ui/pages/artist";
import LoginPage from "@/lib/shared/views/ui/pages/login";
import { MainPage } from "@/lib/shared/views/ui/pages/main";
import { NotFoundPage } from "@/lib/shared/views/ui/pages/not-found";
import { PlaylistPage } from "@/lib/shared/views/ui/pages/playlist";
import { SearchPage } from "@/lib/shared/views/ui/pages/search";
import SettingsPage from "@/lib/shared/views/ui/pages/settings";
import { UserPage } from "@/lib/shared/views/ui/pages/user";
import { YouTubeSearchPage } from "@/lib/shared/views/ui/pages/youtube-search";

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

	if (!known) {
		return (
			<div class="h-dvh flex flex-col bg-stone-950">
				<div
					style="--wails-draggable: drag"
					class="h-10 border-b border-zinc-800 bg-zinc-950/80 backdrop-blur shrink-0"
				/>
				<div class="flex flex-1 items-center justify-center">
					<span class="text-zinc-500">…</span>
				</div>
			</div>
		);
	}

	if (!connected) return <LoginPage />;

	return children;
}

function Router() {
	return (
		<QueryClientProvider client={queryClient}>
			<SpotifyGate>
				<Switch>
					<Route path="/playlist/:id" component={PlaylistPage} />
					<Route path="/artist/:id" component={ArtistPage} />
					<Route path="/album/:id" component={AlbumPage} />
					<Route path="/search/youtube/:query" component={YouTubeSearchPage} />
					<Route path="/search/:type/:query" component={SearchPage} />
					<Route path="/settings" component={SettingsPage} />
					<Route path="/:user" component={UserPage} />
					<Route path="/" component={MainPage} />
					<Route component={NotFoundPage} />
				</Switch>
			</SpotifyGate>
		</QueryClientProvider>
	);
}

export default Router;
