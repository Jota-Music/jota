import { QueryClient, QueryClientProvider } from "@tanstack/preact-query";
import { Route, Switch } from "wouter-preact";
import { MainPage } from "@/lib/shared/views/ui/pages/main";
import { PlaylistPage } from "@/lib/shared/views/ui/pages/playlist";
import RegisterPage from "@/lib/shared/views/ui/pages/register";
import SettingsPage from "@/lib/shared/views/ui/pages/settings";
import { UserPage } from "@/lib/shared/views/ui/pages/user";

const queryClient = new QueryClient();

function Router() {
	return (
		<QueryClientProvider client={queryClient}>
			<Switch>
				<Route path="/playlist/:id" component={PlaylistPage} />
				<Route path="/register" component={RegisterPage} />
				<Route path="/settings" component={SettingsPage} />
				<Route path="/:user" component={UserPage} />
				<Route path="/" component={MainPage} />
				<Route>404: No such page!</Route>
			</Switch>
		</QueryClientProvider>
	);
}

export default Router;
