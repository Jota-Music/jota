import { Loader } from "lucide-preact";
import type { ComponentChildren } from "preact";
import { useState } from "preact/hooks";
import {
	loginSpotifyAndWait,
	spotifyConnected,
} from "@/lib/auth/views/stores/session";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";
import DefaultLayout from "@/lib/shared/views/ui/layouts/default";

export function SpotifyConnect() {
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState("");

	async function handleLogin() {
		setBusy(true);
		setError("");
		const ok = await loginSpotifyAndWait();
		if (!ok) {
			setBusy(false);
			setError("Could not connect to Spotify.");
		}
	}

	if (busy) {
		return (
			<div class="flex flex-col items-center gap-3 text-center">
				<Loader size={24} class="animate-spin text-zinc-500" />
				<p class="text-sm text-zinc-500">
					Complete Spotify login, then return here.
				</p>
			</div>
		);
	}

	return (
		<div class="flex flex-col items-center gap-3 text-center">
			<SpotifyIcon size={24} class="text-zinc-400" />
			<p class="text-sm text-zinc-400">Connect with Spotify to use this.</p>
			<button
				type="button"
				onClick={() => void handleLogin()}
				class="rounded-full bg-zinc-800 px-5 py-2 text-sm font-semibold text-zinc-100 transition-colors hover:bg-zinc-700 cursor-pointer"
			>
				Connect with Spotify
			</button>
			{error && <p class="text-xs text-red-400">{error}</p>}
		</div>
	);
}

export function RequireSpotify({ children }: { children: ComponentChildren }) {
	if (spotifyConnected.value) return <>{children}</>;
	return (
		<DefaultLayout class="gap-6">
			<div class="flex min-h-0 flex-1 items-center justify-center">
				<SpotifyConnect />
			</div>
		</DefaultLayout>
	);
}
