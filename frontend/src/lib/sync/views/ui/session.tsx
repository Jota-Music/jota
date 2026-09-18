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
import { useRef, useState } from "preact/hooks";
import { useLocation } from "wouter-preact";
import { pendingStart } from "@/lib/music/views/stores/audio";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { ConfirmModal } from "@/lib/shared/views/ui/components/confirm";
import { Modal, ModalHeader } from "@/lib/shared/views/ui/components/modal";
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
			closeLabel={t("sync.session.close")}
			hideClose
		>
			<ModalHeader
				close={() => (showSync.value = false)}
				closeLabel={t("sync.session.close")}
			>
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
				<h2 id="sync-panel-title" class="text-sm font-medium text-white">
					{t("sync.session.title")}
				</h2>
				<span class="text-xs text-zinc-500 tabular-nums">
					{active
						? connected
							? store.joined.value
								? t("sync.session.connected", {
										count: store.peers.value,
									})
								: t("sync.session.joining")
							: t("sync.session.connecting")
						: store.status.value === "connecting"
							? t("sync.session.connecting")
							: t("sync.session.offline")}
				</span>
			</ModalHeader>

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
	const [confirming, setConfirming] = useState(false);

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
					<p class="text-sm font-medium text-zinc-100">
						{t("sync.session.relayRequired")}
					</p>
					<p class="text-xs text-zinc-500">
						{t("sync.session.relayRequiredBody")}
					</p>
				</div>
				<button
					type="button"
					class="flex items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 hover:bg-zinc-700 transition-colors cursor-pointer"
					onClick={openSettings}
				>
					<Settings size={16} />
					{t("sync.session.openSettings")}
				</button>
			</div>
		);
	}

	return (
		<div class="flex flex-col gap-3">
			<label for="sync-room" class="text-xs text-zinc-500">
				{t("sync.session.roomCode")}
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
					{t("sync.session.generate")}
				</button>
			</div>
			<p class="text-xs text-zinc-500">
				{active
					? t("sync.session.inRoom", { room: currentRoom })
					: t("sync.session.hostHint")}
			</p>
			<label for="sync-pass" class="text-xs text-zinc-500">
				{t("sync.session.password")}
			</label>
			<PasswordInput
				id="sync-pass"
				class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all"
				placeholder={t("common.optional")}
				label={t("sync.session.password")}
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
				{connecting
					? t("sync.session.connecting")
					: active
						? t("sync.session.switch")
						: t("sync.session.connect")}
			</button>
			{ready && !savedRoom && (
				<button
					type="button"
					title={t("sync.session.saveTitle")}
					class="flex cursor-pointer items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-zinc-300 transition-colors hover:bg-zinc-800 disabled:opacity-40"
					disabled={saving}
					onClick={save}
				>
					<BookmarkPlus size={16} />
					{saving ? t("common.saving") : t("sync.session.saveRoom")}
				</button>
			)}
			{ready && savedRoom && (
				<div class="flex items-center gap-2">
					<p class="flex flex-1 items-center justify-center gap-2 rounded-lg border border-zinc-800 px-4 py-2.5 text-sm text-green-400">
						<BookmarkCheck size={16} />
						{saving ? t("common.saving") : t("sync.session.saved")}
					</p>
					<button
						type="button"
						title={t("sync.session.removeTitle")}
						class="flex h-11 w-11 shrink-0 cursor-pointer items-center justify-center rounded-lg border border-zinc-800 text-zinc-400 transition-colors hover:bg-zinc-800 hover:text-red-300 disabled:opacity-40"
						disabled={removing}
						onClick={() => setConfirming(true)}
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
					{host ? t("sync.session.stop") : t("sync.session.leave")}
				</button>
			)}
			{pendingStart.value && (
				<p class="text-xs text-zinc-500">{t("sync.session.waiting")}</p>
			)}
			{store.error.value && (
				<p class="text-xs text-red-400">{store.error.value}</p>
			)}
			<p class="pt-1 text-xs text-zinc-600">
				{t("sync.session.relay", { url: relay })}{" "}
				<button
					type="button"
					class="underline hover:text-zinc-400 cursor-pointer"
					onClick={openSettings}
				>
					{t("sync.session.settings")}
				</button>
			</p>

			<ConfirmModal
				open={confirming}
				danger
				pending={removing}
				close={() => setConfirming(false)}
				onConfirm={() => {
					if (savedRoom) unsave(savedRoom.id);
					setConfirming(false);
				}}
			/>
		</div>
	);
}
