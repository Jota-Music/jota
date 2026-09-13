import { OpenURL } from "@bindings/app";
import { Browser } from "@wailsio/runtime";
import { RadioTower, TriangleAlert, UserRound } from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import {
	disconnectSpotify,
	spotifyUser,
} from "@/lib/auth/views/stores/session";
import { cn } from "@/lib/shared/utils/tw";
import useMeta from "@/lib/shared/views/hooks/use-meta";
import { PasswordInput } from "@/lib/shared/views/ui/components/password-input";
import ClearLayout from "@/lib/shared/views/ui/layouts/clear";
import * as transport from "@/lib/sync/app/transport";
import * as store from "@/lib/sync/views/stores";

const inputClass =
	"w-full rounded-lg border border-zinc-800 bg-zinc-950 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all";

function SettingsPage() {
	useMeta("Jota | Settings", "Configure your Jota settings");

	return (
		<ClearLayout>
			<div class="flex flex-1 flex-col gap-6 overflow-y-auto py-6">
				<header class="space-y-1">
					<h1 class="text-xl font-bold text-zinc-100">Settings</h1>
					<p class="text-sm text-zinc-500">
						Manage your account and how Jota syncs playback.
					</p>
				</header>

				<RelaySettings />
				<AccountSettings />

				<footer class="mt-auto pt-6 text-center text-xs text-zinc-600">
					Developed by{" "}
					<button
						type="button"
						onClick={openDeveloper}
						class="cursor-pointer text-zinc-400 underline underline-offset-2 transition-colors hover:text-zinc-200"
					>
						@salvadorsru
					</button>
				</footer>
			</div>
		</ClearLayout>
	);
}

const GITHUB_URL = "https://github.com/salvadorsru";

function openDeveloper(): void {
	if (navigator.userAgent.toLowerCase().includes("android")) {
		void OpenURL(GITHUB_URL);
	} else {
		void Browser.OpenURL(GITHUB_URL);
	}
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
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<RadioTower size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">Listen together</h2>
			</div>
			<p class="text-xs text-zinc-500">
				Relay server that syncs playback between devices. Rooms and their
				passwords are set from the Listen together panel.
			</p>

			<div class="space-y-1.5">
				<div class="flex items-center justify-between">
					<label for="sync-relay" class="text-xs font-medium text-zinc-400">
						Relay server
					</label>
					<RelayStatus state={relay} />
				</div>
				<input
					id="sync-relay"
					class={inputClass}
					placeholder="relay.example.com"
					value={url}
					onInput={(e) => (store.relayUrl.value = e.currentTarget.value)}
				/>
			</div>

			<div class="space-y-1.5">
				<label for="sync-token" class="text-xs font-medium text-zinc-400">
					Auth token
				</label>
				<PasswordInput
					id="sync-token"
					class={inputClass}
					placeholder="optional"
					label="Auth token"
					value={store.token.value}
					onValue={(v) => (store.token.value = v)}
				/>
			</div>

			{store.tokenRequired.value && (
				<p class="flex items-center gap-1.5 text-xs text-yellow-400">
					<TriangleAlert size={14} class="shrink-0" />
					This relay requires an auth token.
				</p>
			)}
		</section>
	);
}

function RelayStatus({
	state,
}: {
	state: "idle" | "checking" | "ok" | "error";
}) {
	if (state === "idle") return null;

	const status = {
		checking: {
			label: "Checking…",
			text: "text-zinc-400",
			dot: "bg-zinc-500 animate-pulse",
		},
		ok: { label: "Reachable", text: "text-green-400", dot: "bg-green-400" },
		error: { label: "Not reachable", text: "text-red-400", dot: "bg-red-400" },
	}[state];

	return (
		<span class={cn("flex items-center gap-1.5 text-xs", status.text)}>
			<span class={cn("size-1.5 rounded-full", status.dot)} />
			{status.label}
		</span>
	);
}

function AccountSettings() {
	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<UserRound size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">Account</h2>
			</div>
			<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
				<div class="min-w-0">
					<p class="text-xs text-zinc-500">Connected as</p>
					<p class="truncate text-sm text-zinc-100">
						{spotifyUser.value ?? "Spotify"}
					</p>
				</div>
				<button
					type="button"
					onClick={() => void disconnectSpotify()}
					class="shrink-0 rounded-lg bg-red-900/50 px-3 py-2 text-xs font-semibold text-red-200 hover:bg-red-900/70 transition-colors cursor-pointer"
				>
					Disconnect
				</button>
			</div>
		</section>
	);
}

export default SettingsPage;
