import {
	Link,
	LogOut,
	Radio,
	RefreshCw,
	ShieldAlert,
	Unplug,
	Users,
	X,
} from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";
import { Sheet } from "@/lib/shared/views/ui/components/sheet";
import * as transport from "@/lib/sync/app/transport";
import * as store from "@/lib/sync/views/stores";
import { showSync } from "@/lib/sync/views/stores";
import "@/lib/sync/views/stores/sync";

export function SyncPanel() {
	const r = store.role.value;
	const status = store.status.value;
	const active = r !== "off";
	const connected = status === "open";

	return (
		<Sheet
			open={showSync.value}
			close={() => (showSync.value = false)}
			labelledBy="sync-panel-title"
			closeLabel="Cerrar Sync"
		>
			<header class="flex shrink-0 flex-col gap-1 border-b border-zinc-800 px-4 py-3">
				<div class="flex items-center justify-between gap-2 text-white">
					<div class="flex items-center gap-2">
						{active ? (
							<Radio
								size={22}
								class={cn(connected ? "text-green-400" : "text-yellow-400")}
							/>
						) : (
							<Users size={22} class="text-zinc-400" />
						)}
						<h2 id="sync-panel-title" class="text-sm font-medium">
							Listen together
						</h2>
						<span class="text-xs text-zinc-500 tabular-nums">
							{active
								? connected
									? `${store.peers.value} connected`
									: "Connecting..."
								: status === "connecting"
									? "Connecting..."
									: "Offline"}
						</span>
					</div>
					<button
						type="button"
						class="rounded-lg p-2 text-zinc-400 transition hover:bg-zinc-800 hover:text-white cursor-pointer"
						aria-label="Cerrar"
						onClick={() => (showSync.value = false)}
					>
						<X size={20} />
					</button>
				</div>
			</header>

			<div class="flex-1 overflow-y-auto px-4 py-4">
				<SessionForm />
			</div>
		</Sheet>
	);
}

function SessionForm() {
	const role = store.role.value;
	const status = store.status.value;
	const [code, setCode] = useState(store.room.value);
	const [relay, setRelay] = useState<"idle" | "checking" | "ok" | "error">(
		"idle",
	);
	const url = store.relayUrl.value;
	const showToken = store.tokenRequired.value;
	const ready = url.trim() !== "" && code.trim() !== "";
	const connecting = status === "connecting";
	const active = role !== "off";

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
		<div class="flex flex-col gap-3">
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
			{showToken && (
				<>
					<p class="flex items-center gap-1.5 text-xs text-yellow-400">
						<ShieldAlert size={14} class="shrink-0" />
						This server requires an auth token.
					</p>
					<input
						id="sync-token"
						type="password"
						class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all"
						placeholder="Insert your token here"
						aria-label="Auth token"
						value={store.token.value}
						onInput={(e) => (store.token.value = e.currentTarget.value)}
					/>
				</>
			)}
			<label for="sync-room" class="text-xs text-zinc-500">
				Room code
			</label>
			<div class="flex gap-2">
				<input
					id="sync-room"
					class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all font-mono"
					placeholder="party-42"
					value={code}
					onInput={(e) => setCode(e.currentTarget.value)}
				/>
				<button
					type="button"
					class="flex shrink-0 items-center gap-2 rounded-lg border border-zinc-800 px-3 text-sm text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
					onClick={() => setCode(randomCode())}
				>
					<RefreshCw size={16} />
					Generate
				</button>
			</div>
			<p class="text-xs text-zinc-500">
				{active
					? "Change the code to move the session to another room."
					: "If nobody is hosting it yet, you become the host; otherwise you join and listen in sync."}
			</p>
			<label for="sync-pass" class="text-xs text-zinc-500">
				Room password
			</label>
			<input
				id="sync-pass"
				type="password"
				class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all"
				placeholder="optional"
				value={store.password.value}
				onInput={(e) => (store.password.value = e.currentTarget.value)}
			/>
			<button
				type="button"
				class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer disabled:opacity-40"
				disabled={!ready || connecting}
				onClick={() => void transport.connect(code.trim())}
			>
				<Link size={16} />
				{connecting ? "Connecting…" : active ? "Change room" : "Connect"}
			</button>
			{active && (
				<button
					type="button"
					class="flex items-center justify-center gap-2 rounded-lg bg-red-900/40 px-4 py-2.5 text-sm text-red-200 hover:bg-red-900/60 transition-colors cursor-pointer"
					onClick={() => transport.stop()}
				>
					{role === "host" ? <Unplug size={16} /> : <LogOut size={16} />}
					{role === "host" ? "Stop session" : "Leave session"}
				</button>
			)}
			{store.error.value && (
				<p class="text-xs text-red-400">{store.error.value}</p>
			)}
		</div>
	);
}

function randomCode(): string {
	return Math.random().toString(36).slice(2, 8);
}
