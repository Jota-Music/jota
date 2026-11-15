import { useSignal } from "@preact/signals";
import { Loader } from "lucide-preact";
import type { ComponentChildren } from "preact";
import {
	cancelSpotifyLogin,
	loginSpotifyAndWait,
	spotifyConnected,
	spotifyDegraded,
	spotifyUser,
} from "@/lib/auth/views/stores/session";
import { Hint } from "@/lib/music/views/ui/shelf/hint";
import { t } from "@/lib/shared/i18n";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";

export function SpotifyConnect() {
	const busy = useSignal(false);
	const error = useSignal("");

	async function handleLogin() {
		busy.value = true;
		error.value = "";
		const ok = await loginSpotifyAndWait();
		busy.value = false;
		if (!ok) error.value = t("auth.spotify.couldNotConnect");
	}

	if (busy.value) {
		return (
			<div class="flex flex-col items-center gap-3 text-center">
				<Loader size={24} class="animate-spin text-zinc-500" />
				<p class="text-balance text-sm text-zinc-500">
					{t("auth.spotify.completeLogin")}
				</p>
				<button
					type="button"
					onClick={() => void cancelSpotifyLogin()}
					class="cursor-pointer text-xs text-zinc-500 underline-offset-2 hover:text-zinc-300 hover:underline"
				>
					{t("common.cancel")}
				</button>
			</div>
		);
	}

	return (
		<div class="flex w-full flex-col items-center gap-3 text-center">
			<p class="text-balance text-xs leading-relaxed text-zinc-500">
				{t("auth.spotify.thirdPartyNote")}
			</p>
			<button
				type="button"
				onClick={() => void handleLogin()}
				class="flex w-full cursor-pointer items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm font-semibold text-zinc-100 transition-colors hover:bg-zinc-700"
			>
				{t("auth.spotify.connect")}
			</button>
			{error.value && (
				<p class="text-balance text-xs text-red-400">{error.value}</p>
			)}
		</div>
	);
}

export function SpotifyHint() {
	return (
		<Hint
			icon={<SpotifyIcon size={24} />}
			class="text-green-500"
			title={t("auth.spotify.hint.title")}
			body={t("auth.spotify.hint.body")}
			steps={[
				t("auth.spotify.hint.step1"),
				t("auth.spotify.hint.step2"),
				t("auth.spotify.hint.step3"),
			]}
			action={<SpotifyConnect />}
		/>
	);
}

export function RequireSpotify({
	children,
	live = false,
}: {
	children: ComponentChildren;
	live?: boolean;
}) {
	if (spotifyConnected.value) return <>{children}</>;
	// Cached routes keep working off the stored catalog, and an account the store
	// already names is one this device has: the restore is still crossing the
	// network, and the home it is about to fill is no reason to show a login
	// panel. Routes marked live (search, radio) have no cache to fall back on, so
	// they stay walled until the session is really there.
	if (!live && (spotifyUser.value || spotifyDegraded.value))
		return <>{children}</>;
	return (
		<div class="flex min-h-0 flex-1 items-center justify-center">
			{spotifyDegraded.value ? <SpotifyUnavailable /> : <SpotifyHint />}
		</div>
	);
}

function SpotifyUnavailable() {
	return (
		<div class="flex flex-col items-center gap-3 text-center">
			<SpotifyIcon size={24} class="text-amber-500/70" />
			<p class="text-balance text-sm text-zinc-400">
				{t("auth.spotify.offlineLimited")}
			</p>
		</div>
	);
}
