import {
	disconnectSpotify,
	spotifyUser,
} from "@/lib/auth/views/stores/session";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import ClearLayout from "@/lib/shared/views/ui/layouts/clear";

function SettingsPage() {
	useMeta("Jota | Settings", "Configure your Jota settings");

	return (
		<ClearLayout class="gap-6">
			<h1 class="text-lg font-bold text-zinc-100">Settings</h1>

			<section class="space-y-3">
				<h2 class="text-sm font-semibold text-zinc-300">YouTube Cookies</h2>
				<p class="text-xs text-zinc-500">
					YouTube cookies are managed via the app config directory.
				</p>
			</section>

			<section class="space-y-3">
				<h2 class="text-sm font-semibold text-zinc-300">Account</h2>
				<p class="text-xs text-zinc-500">
					Connected as {spotifyUser.value ?? "Spotify"}
				</p>
				<button
					type="button"
					onClick={() => void disconnectSpotify()}
					class="flex items-center gap-1.5 rounded-lg bg-red-900/60 px-4 py-2 text-sm font-bold text-red-200 hover:bg-red-800/60 transition-colors cursor-pointer"
				>
					Disconnect Spotify
				</button>
			</section>
		</ClearLayout>
	);
}

export default SettingsPage;
