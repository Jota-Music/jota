import {
	BookmarkPlus,
	LogIn,
	RefreshCw,
	Settings,
	Turntable,
} from "lucide-preact";
import { useEffect, useRef } from "preact/hooks";
import { useLocation } from "wouter-preact";
import { pendingStart } from "@/lib/music/views/stores/audio";
import { t } from "@/lib/shared/i18n";
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
		>
			<ModalHeader>
				<Turntable
					size={22}
					class={
						active
							? connected
								? "text-green-400"
								: "text-yellow-400"
							: "text-zinc-400"
					}
				/>
				<h2 id="sync-panel-title" class="text-sm font-medium text-white">
					{t("sync.session.title")}
				</h2>
				<span class="min-w-0 truncate text-xs text-zinc-500 tabular-nums">
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

			<div class="relative min-h-0 shrink">
				<div
					ref={listRef}
					class="max-h-[min(70dvh,26rem)] overflow-y-auto px-4 py-4"
				>
					<SessionForm />
				</div>
				<Scrollbar target={listRef} gutter={16} />
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
		saving,
		save,
		join,
	} = useRoomForm();
	const [, setLocation] = useLocation();
	const field = useRef<HTMLInputElement>(null);

	// The panel opens ready to type: the code is the reason it is here, and a
	// focused, selected field lets the reader rewrite it or press Enter without a
	// second tap.
	useEffect(() => {
		field.current?.focus();
		field.current?.select();
	}, []);

	const openSettings = () => {
		showSync.value = false;
		setLocation("/settings");
	};

	// Enter saves what is typed and closes: naming a room and entering it are
	// two gestures, and only the buttons below or the shelf enter it.
	const submit = () => {
		if (!ready || saving) return;
		save();
		showSync.value = false;
	};

	// The shortcut skips the shelf: the room is entered with what is typed, and
	// it shows up there as the room we are in, unsaved, until the save button on
	// its row keeps it.
	const enter = () => {
		if (!ready) return;
		join();
		showSync.value = false;
	};

	if (relay === "") {
		return (
			<div class="flex flex-col items-center gap-4 py-4 text-center">
				<div class="flex size-14 items-center justify-center rounded-full bg-zinc-900 ring-1 ring-zinc-800">
					<Turntable size={26} class="text-zinc-500" />
				</div>
				<div class="space-y-1">
					<p class="text-sm font-medium text-zinc-100">
						{t("sync.session.relayRequired")}
					</p>
					<p class="mb-2 block text-xs text-balance text-zinc-500">
						{t("sync.session.relayRequiredBody")}
					</p>
				</div>
				<button
					type="button"
					class="flex cursor-pointer items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 transition-colors hover:bg-zinc-700"
					onClick={openSettings}
				>
					<Settings size={16} />
					{t("sync.session.openSettings")}
				</button>
			</div>
		);
	}

	return (
		<div class="flex flex-col gap-2">
			<div class="relative">
				<input
					ref={field}
					aria-label={t("sync.session.roomCode")}
					class="w-full rounded-lg border border-zinc-800 bg-zinc-900 py-3 pr-11 pl-3 font-mono text-sm text-zinc-100 outline-none transition-all focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
					placeholder="party-42"
					value={code}
					onInput={(e) => setCode(e.currentTarget.value)}
					onKeyDown={(e) => {
						if (e.key === "Enter") submit();
					}}
				/>
				<button
					type="button"
					title={t("sync.session.generate")}
					aria-label={t("sync.session.generate")}
					onClick={generate}
					class="absolute top-1/2 right-1.5 -translate-y-1/2 cursor-pointer rounded-md p-1.5 text-zinc-500 transition-colors hover:text-zinc-200"
				>
					<RefreshCw size={16} />
				</button>
			</div>
			<PasswordInput
				class="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-3 text-sm text-zinc-100 outline-none transition-all focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
				placeholder={t("sync.session.password")}
				label={t("sync.session.password")}
				value={password}
				onValue={setPassword}
				onBlur={commitPassword}
				onEnter={submit}
			/>
			<div class="grid gap-2 sm:grid-cols-2">
				<button
					type="button"
					class="flex cursor-pointer items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 transition-colors hover:bg-zinc-700 disabled:opacity-40"
					disabled={!ready}
					onClick={enter}
				>
					<LogIn size={16} />
					{t("sync.session.join")}
				</button>
				<button
					type="button"
					class="flex cursor-pointer items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 transition-colors hover:bg-zinc-700 disabled:opacity-40"
					disabled={!ready || saving}
					onClick={submit}
				>
					<BookmarkPlus size={16} />
					{t("common.save")}
				</button>
			</div>
			{pendingStart.value && (
				<p class="text-xs text-balance text-zinc-500">
					{t("sync.session.waiting")}
				</p>
			)}
			{store.error.value && (
				<p class="text-xs text-red-400">{store.error.value}</p>
			)}
			<p class="flex items-center gap-1 pt-1 text-xs text-zinc-600">
				<span class="min-w-0 truncate">
					{t("sync.session.relay", { url: relay })}
				</span>
				<button
					type="button"
					class="shrink-0 cursor-pointer underline hover:text-zinc-400"
					onClick={openSettings}
				>
					{t("sync.session.settings")}
				</button>
			</p>
		</div>
	);
}
