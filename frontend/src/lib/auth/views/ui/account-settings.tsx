import { useSignal } from "@preact/signals";
import { UserRound } from "lucide-preact";
import {
	disconnectSpotify,
	spotifyConnected,
	spotifyDegraded,
	spotifyUser,
} from "@/lib/auth/views/stores/session";
import {
	exportYouTubeCookiesFile,
	signOutYouTube,
	youtubeAccount,
	youtubeSessionFull,
	youtubeSignedIn,
} from "@/lib/auth/views/stores/youtube";
import { SpotifyConnect } from "@/lib/auth/views/ui/spotify-connect";
import {
	BrowserImport,
	YouTubeSignIn,
} from "@/lib/auth/views/ui/youtube-sign-in";
import { t } from "@/lib/shared/i18n";
import { translateError } from "@/lib/shared/i18n/errors";

export function AccountSettings() {
	const exportBusy = useSignal(false);
	const error = useSignal<string | null>(null);

	async function doExport() {
		exportBusy.value = true;
		error.value = null;
		try {
			await exportYouTubeCookiesFile();
		} catch (err) {
			error.value = translateError(err);
		} finally {
			exportBusy.value = false;
		}
	}

	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<UserRound size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("auth.account.title")}
				</h2>
			</div>
			{spotifyConnected.value || spotifyDegraded.value ? (
				<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
					<div class="min-w-0">
						<p class="text-xs text-zinc-500">
							{spotifyDegraded.value
								? t("auth.account.cachedAs")
								: t("auth.account.connectedAs")}
						</p>
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
					<div class="space-y-2">
						<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
							<div class="min-w-0">
								{youtubeAccount.value && (
									<p class="text-xs text-zinc-500">
										{t("auth.account.connectedAs")}
									</p>
								)}
								<p class="truncate text-sm text-zinc-100">
									{youtubeAccount.value || t("auth.account.signedIn")}
								</p>
							</div>
							<button
								type="button"
								onClick={() => void signOutYouTube()}
								class="shrink-0 rounded-lg bg-red-900/50 px-3 py-2 text-xs font-semibold text-red-200 hover:bg-red-900/70 transition-colors cursor-pointer"
							>
								{t("auth.account.signOut")}
							</button>
						</div>

						{!youtubeSessionFull.value && (
							<div class="space-y-3 rounded-xl border border-amber-500/40 bg-amber-500/10 p-3">
								<p class="text-xs text-amber-300">
									{t("auth.youtube.partial")}
								</p>
								<BrowserImport />
							</div>
						)}

						<button
							type="button"
							onClick={() => void doExport()}
							disabled={exportBusy.value}
							class="flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-lg bg-(--dominant-color) px-3 py-2 text-xs font-semibold text-(--binary-color) transition-opacity hover:opacity-85 disabled:cursor-not-allowed disabled:opacity-50"
						>
							{exportBusy.value
								? t("common.saving")
								: t("auth.account.exportCookies")}
						</button>

						{error.value && <p class="text-xs text-red-400">{error.value}</p>}
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
