import {
	Download,
	ExternalLink,
	Info,
	Loader2,
	RadioTower,
	TriangleAlert,
	UserRound,
} from "lucide-preact";
import { useEffect, useRef, useState } from "preact/hooks";
import {
	disconnectSpotify,
	spotifyConnected,
	spotifyUser,
} from "@/lib/auth/views/stores/session";
import { SpotifyConnect } from "@/lib/auth/views/ui/spotify-connect";
import { open } from "@/lib/shared/utils/open";
import { cn } from "@/lib/shared/utils/tw";
import { PasswordInput } from "@/lib/shared/views/ui/components/password-input";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";
import ClearLayout from "@/lib/shared/views/ui/layouts/clear";
import * as transport from "@/lib/sync/app/transport";
import * as store from "@/lib/sync/views/stores";
import {
	install,
	installing,
	percent,
	update,
	version,
} from "@/lib/update/views/stores/update";

const inputClass =
	"w-full rounded-lg border border-zinc-800 bg-zinc-950 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all";

function SettingsPage() {
	const listRef = useRef<HTMLDivElement>(null);

	return (
		<ClearLayout>
			<div class="relative min-h-0 flex-1">
				<div
					ref={listRef}
					class="h-full flex flex-col gap-6 overflow-y-auto py-6"
				>
					<header class="space-y-1">
						<h1 class="text-xl font-bold text-zinc-100">Settings</h1>
						<p class="text-sm text-zinc-500">
							Manage your account and how Jota syncs playback.
						</p>
					</header>

					<AccountSettings />
					<RelaySettings />
					<AboutSettings />

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
				<Scrollbar target={listRef} />
			</div>
		</ClearLayout>
	);
}

const GITHUB_URL = "https://github.com/salvadorsru";

function openDeveloper(): void {
	void open(GITHUB_URL);
}

function AboutSettings() {
	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<Info size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">About</h2>
			</div>

			<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
				<div class="min-w-0">
					<p class="text-xs text-zinc-500">Version</p>
					<p class="truncate text-sm text-zinc-100">{version.value || "dev"}</p>
				</div>
				<UpdateAction />
			</div>
		</section>
	);
}

function UpdateAction() {
	const info = update.value;

	if (!info?.available) {
		return null;
	}

	const p = installing.value ? percent() : null;

	return (
		<button
			type="button"
			disabled={installing.value}
			onClick={() => (info.installable ? void install() : void open(info.url))}
			class="flex shrink-0 cursor-pointer items-center gap-1.5 rounded-lg bg-(--dominant-color) px-3 py-2 text-xs font-semibold text-(--binary-color) transition-opacity hover:opacity-85 disabled:opacity-50"
		>
			{installing.value ? (
				<Loader2 size={12} class="animate-spin" />
			) : info.installable ? (
				<Download size={12} />
			) : (
				<ExternalLink size={12} />
			)}
			{installing.value
				? p == null
					? "Updating…"
					: `Updating… ${p}%`
				: info.installable
					? `Update to ${info.latest}`
					: `Download ${info.latest}`}
		</button>
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

			<div
				class="grid transition-all duration-200 ease-out"
				style={{
					gridTemplateRows: store.tokenRequired.value ? "1fr" : "0fr",
				}}
			>
				<div class="min-h-0 overflow-hidden">
					<div class="space-y-1.5 pt-1.5">
						<label for="sync-token" class="text-xs font-medium text-zinc-400">
							Auth token
						</label>
						<PasswordInput
							id="sync-token"
							class={inputClass}
							label="Auth token"
							value={store.token.value}
							onValue={(v) => (store.token.value = v)}
						/>
						<p class="flex items-center gap-1.5 text-xs text-yellow-400">
							<TriangleAlert size={14} class="shrink-0" />
							This relay requires an auth token.
						</p>
					</div>
				</div>
			</div>
		</section>
	);
}

function RelayStatus({
	state,
}: {
	state: "idle" | "checking" | "ok" | "error";
}) {
	const status = {
		idle: null,
		checking: {
			label: "Checking…",
			text: "text-zinc-400",
			dot: "bg-zinc-500 animate-pulse",
		},
		ok: { label: "Reachable", text: "text-green-400", dot: "bg-green-400" },
		error: { label: "Not reachable", text: "text-red-400", dot: "bg-red-400" },
	}[state];

	if (!status) {
		return <span class="h-4 w-24" aria-hidden />;
	}

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
			{spotifyConnected.value ? (
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
			) : (
				<div class="rounded-xl bg-zinc-950 p-6 ring-1 ring-zinc-800">
					<SpotifyConnect />
				</div>
			)}
		</section>
	);
}

export default SettingsPage;
