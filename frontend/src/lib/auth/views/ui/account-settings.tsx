import { UserRound } from "lucide-preact";
import {
	disconnectSpotify,
	spotifyConnected,
	spotifyUser,
} from "@/lib/auth/views/stores/session";
import {
	signOutYouTube,
	youtubeSignedIn,
} from "@/lib/auth/views/stores/youtube";
import { SpotifyConnect } from "@/lib/auth/views/ui/spotify-connect";
import { YouTubeSignIn } from "@/lib/auth/views/ui/youtube-sign-in";
import { t } from "@/lib/shared/i18n";

export function AccountSettings() {
	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<UserRound size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("auth.account.title")}
				</h2>
			</div>
			{spotifyConnected.value ? (
				<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
					<div class="min-w-0">
						<p class="text-xs text-zinc-500">{t("auth.account.connectedAs")}</p>
						<p class="truncate text-sm text-zinc-100">
							{spotifyUser.value ?? "Spotify"}
						</p>
					</div>
					<button
						type="button"
						onClick={() => void disconnectSpotify()}
						class="shrink-0 rounded-lg bg-red-900/50 px-3 py-2 text-xs font-semibold text-red-200 hover:bg-red-900/70 transition-colors cursor-pointer"
					>
						{t("auth.account.disconnect")}
					</button>
				</div>
			) : (
				<div class="rounded-xl bg-zinc-950 p-6 ring-1 ring-zinc-800">
					<SpotifyConnect />
				</div>
			)}

			<div class="space-y-2 border-t border-zinc-800 pt-4">
				<p class="text-xs text-zinc-500">{t("auth.account.youtubeHint")}</p>
				{youtubeSignedIn.value ? (
					<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
						<p class="text-sm text-zinc-100">{t("auth.account.signedIn")}</p>
						<button
							type="button"
							onClick={() => void signOutYouTube()}
							class="shrink-0 rounded-lg bg-red-900/50 px-3 py-2 text-xs font-semibold text-red-200 hover:bg-red-900/70 transition-colors cursor-pointer"
						>
							{t("auth.account.signOut")}
						</button>
					</div>
				) : (
					<div class="rounded-xl bg-zinc-950 p-4 ring-1 ring-zinc-800">
						<YouTubeSignIn />
					</div>
				)}
			</div>
		</section>
	);
}
