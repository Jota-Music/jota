import {
	BookmarkCheck,
	BookmarkPlus,
	LogIn,
	LogOut,
	RefreshCw,
	Settings,
	Trash2,
	Turntable,
	Unplug,
} from "lucide-preact";
import { useRef } from "preact/hooks";
import { useLocation } from "wouter-preact";
import { pendingStart } from "@/lib/music/views/stores/audio";
import { cn } from "@/lib/shared/utils/tw";
import { Modal } from "@/lib/shared/views/ui/components/modal";
import { PasswordInput } from "@/lib/shared/views/ui/components/password-input";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";
import { useRoomForm } from "@/lib/sync/views/hooks/use-room-form";
import * as store from "@/lib/sync/views/stores";
import { showSync } from "@/lib/sync/views/stores";
import "@/lib/sync/views/stores/room";

export function SyncPanel() {
	const active = store.role.value !== "off";
	const connected = store.status.value === "open";
	const listRef = useRef<HTMLDivElement>(null);

	return (
		<Modal
			open={showSync.value}
			close={() => (showSync.value = false)}
			labelledBy="sync-panel-title"
			closeLabel="Close Sync"
		>
			<header class="flex shrink-0 flex-col gap-1 border-b border-zinc-800 px-4 py-3">
				<div class="flex items-center justify-between gap-2 text-white">
					<div class="flex items-center gap-2">
						<Turntable
							size={22}
							class={cn(
								active
									? connected
										? "text-green-400"
										: "text-yellow-400"
									: "text-zinc-400",
							)}
						/>
						<h2 id="sync-panel-title" class="text-sm font-medium">
							Rooms
						</h2>
						<span class="text-xs text-zinc-500 tabular-nums">
							{active
								? connected
									? store.joined.value
										? `${store.peers.value} connected`
										: "Joining..."
									: "Connecting..."
								: store.status.value === "connecting"
									? "Connecting..."
									: "Offline"}
						</span>
					</div>
				</div>
			</header>

			<div class="relative h-[min(70dvh,26rem)]">
				<div ref={listRef} class="h-full overflow-y-auto px-4 py-4">
					<SessionForm />
				</div>
				<Scrollbar target={listRef} />
			</div>
		</Modal>
	);
}

function SessionForm() {
	const {
		code,
		setCode,
		generate,
		password,
		setPassword,
		commitPassword,
		relay,
		ready,
		connecting,
		active,
		currentRoom,
		switching,
		savedRoom,
		host,
		saving,
		removing,
		connect,
		leave,
		save,
		unsave,
	} = useRoomForm();
	const [, setLocation] = useLocation();

	const openSettings = () => {
		showSync.value = false;
		setLocation("/settings");
	};

	if (relay === "") {
		return (
			<div class="flex h-full flex-col items-center justify-center gap-4 text-center">
				<div class="flex size-14 items-center justify-center rounded-full bg-zinc-900 ring-1 ring-zinc-800">
					<Turntable size={26} class="text-zinc-500" />
				</div>
				<div class="space-y-1">
					<p class="text-sm font-medium text-zinc-100">Relay server required</p>
					<p class="text-xs text-zinc-500">
						To use this feature you need a relay server. Configure one in
						Settings to start or join a room.
					</p>
				</div>
				<button
					type="button"
					class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer"
					onClick={openSettings}
				>
					<Settings size={16} />
					Open settings
				</button>
			</div>
		);
	}

	return (
		<div class="flex flex-col gap-3">
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
					onClick={generate}
				>
					<RefreshCw size={16} />
					Generate
				</button>
			</div>
			<p class="text-xs text-zinc-500">
				{active ? (
					<>
						You are in room{" "}
						<span class="font-mono text-zinc-300">{currentRoom}</span>. Edit the
						code to switch rooms.
					</>
				) : (
					"If nobody is hosting it yet, you become the host; otherwise you join and listen in sync."
				)}
			</p>
			<label for="sync-pass" class="text-xs text-zinc-500">
				Room password
			</label>
			<PasswordInput
				id="sync-pass"
				class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all"
				placeholder="Optional"
				label="Room password"
				value={password}
				onValue={setPassword}
				onBlur={commitPassword}
			/>
			<button
				type="button"
				class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer disabled:opacity-40"
				disabled={!ready || connecting || (active && !switching)}
				onClick={connect}
			>
				<LogIn size={16} />
				{connecting ? "Connecting…" : active ? "Switch room" : "Connect"}
			</button>
			{ready && !savedRoom && (
				<button
					type="button"
					title="Save this room to the Rooms shelf"
					class="flex cursor-pointer items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-300 transition-colors hover:bg-zinc-800 disabled:opacity-40"
					disabled={saving}
					onClick={save}
				>
					<BookmarkPlus size={16} />
					{saving ? "Saving…" : "Save room"}
				</button>
			)}
			{ready && savedRoom && (
				<div class="flex items-center gap-2">
					<p class="flex flex-1 items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-green-400">
						<BookmarkCheck size={16} />
						{saving ? "Saving…" : "Saved to Rooms"}
					</p>
					<button
						type="button"
						title="Remove from Rooms"
						class="flex h-11 w-11 shrink-0 cursor-pointer items-center justify-center rounded-lg border border-zinc-800 text-zinc-400 transition-colors hover:bg-zinc-800 hover:text-red-300 disabled:opacity-40"
						disabled={removing}
						onClick={() => unsave(savedRoom.id)}
					>
						<Trash2 size={16} />
					</button>
				</div>
			)}
			{active && (
				<button
					type="button"
					class="flex items-center justify-center gap-2 rounded-lg bg-red-900/40 px-4 py-2.5 text-sm text-red-200 hover:bg-red-900/60 transition-colors cursor-pointer"
					onClick={leave}
				>
					{host ? <Unplug size={16} /> : <LogOut size={16} />}
					{host ? "Stop session" : "Leave session"}
				</button>
			)}
			{pendingStart.value && (
				<p class="text-xs text-zinc-500">Waiting for the room to load…</p>
			)}
			{store.error.value && (
				<p class="text-xs text-red-400">{store.error.value}</p>
			)}
			<p class="pt-1 text-xs text-zinc-600">
				Relay: <span class="text-zinc-400">{relay}</span>{" "}
				<button
					type="button"
					class="underline hover:text-zinc-400 cursor-pointer"
					onClick={openSettings}
				>
					Settings
				</button>
			</p>
		</div>
	);
}
