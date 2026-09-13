import { useEffect, useState } from "preact/hooks";
import {
	disconnectSpotify,
	spotifyUser,
} from "@/lib/auth/views/stores/session";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import ClearLayout from "@/lib/shared/views/ui/layouts/clear";
import * as transport from "@/lib/sync/app/transport";
import * as store from "@/lib/sync/views/stores";

function SettingsPage() {
	useMeta("Jota | Settings", "Configure your Jota settings");

	return (
		<ClearLayout class="gap-6">
			<h1 class="text-lg font-bold text-zinc-100">Settings</h1>

			<RelaySettings />

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

function RelaySettings() {
	const url = store.relayUrl.value;
	const [relay, setRelay] = useState<"idle" | "checking" | "ok" | "error">(
		"idle",
	);

	useEffect(() => {
		const trimmed = url.trim();
		if (trimmed === "") {
			setRelay("idle");
			store.tokenRequired.value = false;
			return;
		}
		let alive = true;
		setRelay("checking");
		const timer = setTimeout(() => {
			transport
				.check(trimmed)
				.then((required) => {
					if (!alive) return;
					setRelay("ok");
					store.tokenRequired.value = required;
				})
				.catch(() => alive && setRelay("error"));
		}, 500);
		return () => {
			alive = false;
			clearTimeout(timer);
		};
	}, [url]);

	return (
		<section class="space-y-3">
			<h2 class="text-sm font-semibold text-zinc-300">Listen together</h2>
			<p class="text-xs text-zinc-500">
				Relay server used to sync playback between devices. Rooms and their
				passwords are set from the Listen together panel.
			</p>
			<label for="sync-relay" class="text-xs text-zinc-500">
				Relay server
			</label>
			<input
				id="sync-relay"
				class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all"
				placeholder="relay.example.com"
				value={url}
				onInput={(e) => (store.relayUrl.value = e.currentTarget.value)}
			/>
			{relay === "checking" && (
				<p class="text-xs text-zinc-500">Checking relay…</p>
			)}
			{relay === "ok" && <p class="text-xs text-green-400">Relay reachable</p>}
			{relay === "error" && (
				<p class="text-xs text-red-400">Relay not reachable</p>
			)}
			<label for="sync-token" class="text-xs text-zinc-500">
				Auth token
			</label>
			<input
				id="sync-token"
				type="password"
				class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all"
				placeholder="optional"
				value={store.token.value}
				onInput={(e) => (store.token.value = e.currentTarget.value)}
			/>
			{store.tokenRequired.value && (
				<p class="text-xs text-yellow-400">
					This relay requires an auth token.
				</p>
			)}
		</section>
	);
}

export default SettingsPage;
