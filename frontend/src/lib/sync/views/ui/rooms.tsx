import { useSignal } from "@preact/signals";
import {
	useMutation,
	useQueries,
	useQuery,
	useQueryClient,
} from "@tanstack/preact-query";
import {
	BookmarkPlus,
	Loader,
	Lock,
	LogIn,
	LogOut,
	Plus,
	PowerOff,
	TriangleAlert,
	Turntable,
	Unplug,
	Wifi,
	WifiOff,
} from "lucide-preact";
import { useEffect } from "preact/hooks";
import { shareLink } from "@/lib/music/views/ui/share";
import { Hint, type Item, RelayHint, Shelf } from "@/lib/music/views/ui/shelf";
import { getOrder, saveOrder } from "@/lib/shared/app/order";
import { applyOrder, move } from "@/lib/shared/app/reorder";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { ConfirmModal } from "@/lib/shared/views/ui/components/confirm";
import type { Action } from "@/lib/shared/views/ui/components/context-menu";
import { roomLink } from "@/lib/sync/app/room-link";
import {
	listRooms,
	type Room,
	type RoomStatus,
	randomCode,
	removeRoom,
	roomStatus,
	saveRoom,
} from "@/lib/sync/app/rooms";
import * as transport from "@/lib/sync/app/transport";
import * as store from "@/lib/sync/views/stores";

const REFRESH_MS = 15000;

// The id of the row for a room we are in but have not saved.
const ACTIVE_ID = "active-room";

type Live = "idle" | "connecting" | "connected";

function Status({
	status,
	pending = false,
	size,
	live,
	members,
	needsPassword,
}: {
	status?: RoomStatus;
	pending?: boolean;
	size: number;
	live: Live;
	members: number;
	needsPassword: boolean;
}) {
	const count = (
		<span
			class="tabular-nums"
			style={{ fontSize: `${Math.round(size * 0.5)}px`, lineHeight: 1 }}
		>
			{members}
		</span>
	);

	if (live === "connecting") {
		return (
			<span title={t("sync.rooms.status.connecting")}>
				<Loader size={24} class="animate-spin text-amber-400" />
			</span>
		);
	}

	if (live === "connected") {
		return (
			<span
				title={t("sync.rooms.status.connected", { count: members })}
				class="relative flex flex-col items-center gap-0.5 text-green-400"
			>
				<Wifi size={size} />
				{count}
				{needsPassword && (
					<Lock
						size={lockSize(size)}
						class="absolute -bottom-0.5 -right-1 text-amber-400"
					/>
				)}
			</span>
		);
	}

	if (pending) {
		return (
			<span title={t("sync.rooms.status.checking")}>
				<PowerOff size={size} class="text-zinc-500" />
			</span>
		);
	}

	if (!status) {
		return (
			<span title={t("sync.rooms.status.unavailable")}>
				<WifiOff size={size} class="text-amber-400" />
			</span>
		);
	}

	if (!status.active) {
		return (
			<span title={t("sync.rooms.status.offline")}>
				<PowerOff size={size} class="text-zinc-500" />
			</span>
		);
	}

	// A room with members but no host is winding down, so warn instead of
	// showing the usual live green.
	const title = status.hasHost
		? t("sync.rooms.status.listening", { count: status.members })
		: t("sync.rooms.status.hostLeft");
	return (
		<span
			title={title}
			class={cn(
				"relative flex flex-col items-center gap-0.5",
				status.hasHost ? "text-green-400" : "text-amber-400",
			)}
		>
			{status.hasHost ? <Wifi size={size} /> : <TriangleAlert size={size} />}
			{count}
			{needsPassword && (
				<Lock
					size={lockSize(size)}
					class="absolute -bottom-0.5 -right-1 text-amber-400"
				/>
			)}
		</span>
	);
}

function lockSize(size: number): number {
	return Math.max(10, Math.round(size * 0.45));
}

// A relay is set, so a room can exist: the shelf is empty only because none has
// been saved yet. The hint says how, instead of leaving the reader with a line
// of grey text and a button above it whose purpose is not obvious.
function RoomsHint({ onNew }: { onNew: () => void }) {
	return (
		<Hint
			icon={<Turntable size={24} />}
			class="text-sky-500"
			title={t("sync.rooms.emptyHint.title")}
			body={t("sync.rooms.emptyHint.body")}
			steps={[
				t("sync.rooms.emptyHint.step1"),
				t("sync.rooms.emptyHint.step2"),
				t("sync.rooms.emptyHint.step3"),
			]}
			action={
				<button
					type="button"
					onClick={onNew}
					class="flex cursor-pointer items-center justify-center gap-2 rounded-lg bg-zinc-800 px-4 py-2.5 text-sm text-zinc-100 transition-colors hover:bg-zinc-700"
				>
					<Plus size={16} />
					{t("sync.rooms.create")}
				</button>
			}
		/>
	);
}

export function RoomsShelf() {
	const queryClient = useQueryClient();
	const confirming = useSignal<string | null>(null);
	const rooms = useQuery({ queryKey: ["rooms"], queryFn: listRooms });
	const orderQuery = useQuery({
		queryKey: ["shelf-order", "rooms"],
		queryFn: () => getOrder("rooms", ""),
	});

	const list = rooms.data ?? [];
	const locked = store.relayLocked.value;
	// The settings form writes this signal untrimmed, so a field holding only
	// spaces is still "no relay".
	const relay = store.relayUrl.value.trim();
	const status = store.status.value;
	const role = store.role.value;
	const current = store.room.value;

	const activeCode = current && status !== "idle" ? current : "";
	const connectingCode =
		status === "connecting" || status === "closed" ? current : "";
	const connectedCode = status === "open" && role !== "off" ? current : "";
	const hosting = role === "host";
	// A session belongs to the shelf while it lasts: the way out sits in the bar
	// next to the rooms it is playing to, not inside the panel that only sets it.
	const session = {
		label: hosting ? t("sync.session.stop") : t("sync.session.leave"),
		icon: hosting ? Unplug : LogOut,
	};

	// Status is read-only: it asks the relay by code and never joins, so polling
	// a shelf of rooms cannot disturb any room.
	const statuses = useQueries({
		queries: list.map((room) => ({
			queryKey: ["room-status", room.id, locked],
			queryFn: () =>
				roomStatus(
					locked ? store.relayUrl.value : room.relayUrl,
					room.code,
					locked ? store.token.value : room.token,
				),
			refetchInterval: REFRESH_MS,
			retry: false,
		})),
	});
	const statusById = new Map(list.map((room, i) => [room.id, statuses[i]]));

	// Refetch right away when we join or leave, instead of waiting for the poll.
	useEffect(() => {
		void queryClient.invalidateQueries({ queryKey: ["room-status"] });
	}, [queryClient, status, role, current]);

	const remove = useMutation({
		mutationFn: removeRoom,
		onSuccess: (next) => queryClient.setQueryData(["rooms"], next),
	});

	const save = useMutation({
		mutationFn: saveRoom,
		onSuccess: (next) => queryClient.setQueryData(["rooms"], next),
	});

	const persist = (room: Room) =>
		save.mutate({
			id: "",
			code: room.code,
			relayUrl: room.relayUrl,
			token: room.token,
			password: room.password,
			name: room.code,
			savedAt: 0,
		});

	const activeSaved = list.find((room) => room.code === activeCode);
	const ephemeral: Room | null =
		activeCode && !activeSaved
			? {
					id: ACTIVE_ID,
					code: activeCode,
					relayUrl: store.relayUrl.value,
					token: store.token.value,
					password: store.password.value,
					name: activeCode,
					savedAt: 0,
				}
			: null;

	const rows: Room[] = ephemeral ? [ephemeral, ...list] : list;

	// The room we are in but have not saved leads the shelf and is not part of
	// the saved order, so it is pinned back to the front after ordering.
	const sorted: Room[] = applyOrder(rows, orderQuery.data ?? []);
	const ordered: Room[] = ephemeral
		? [ephemeral, ...sorted.filter((room) => room.id !== ACTIVE_ID)]
		: sorted;

	const apply = (room: Room) => {
		if (!locked) {
			store.relayUrl.value = room.relayUrl;
			store.token.value = room.token;
		}
		store.password.value = room.password;
		store.error.value = "";
	};

	const open = (id: string) => {
		const room = rows.find((r) => r.id === id);
		if (!room) return;
		apply(room);
		store.draft.value = room.code;
		store.showSync.value = true;
	};

	const join = (room: Room) => {
		apply(room);
		void transport.connect(room.code);
	};

	// Creating opens the panel on a fresh code: the room exists once the listener
	// confirms it there, and Enter is that confirmation.
	const create = () => {
		store.draft.value = randomCode();
		store.showSync.value = true;
	};

	const leave = () => {
		transport.stop();
	};

	const onRemove = (id: string) => {
		confirming.value = id;
	};

	const confirmRemove = () => {
		const id = confirming.value;
		confirming.value = null;
		if (!id) return;
		const room = rows.find((r) => r.id === id);
		// A room we are in is left first, so we never stay in a room the shelf no
		// longer shows.
		if (room && room.code === activeCode) transport.stop();
		remove.mutate(id);
	};

	const items: Item[] = ordered.map((room) => {
		const query = statusById.get(room.id);
		const isActive = room.code === activeCode;
		const live: Live =
			isActive && connectedCode === room.code
				? "connected"
				: isActive && connectingCode === room.code
					? "connecting"
					: "idle";
		const members =
			live === "connected" ? store.peers.value : (query?.data?.members ?? 0);
		const needsPassword = !!query?.data?.locked && !room.password;
		const renderStatus = (size: number) => (
			<Status
				status={query?.data}
				pending={query?.isLoading}
				size={size}
				live={live}
				members={members}
				needsPassword={needsPassword}
			/>
		);
		// The relay caches the room's playback, so a live room can show what it
		// is playing before we join. The status stays overlaid on the cover.
		const song = query?.data?.active ? query.data.state?.song : undefined;
		const cover = song?.album?.covers?.[0];
		const artist = song?.artists?.[0]?.name;
		const connected = live === "connected";
		const menu: Action[] = [
			connected
				? { icon: LogOut, label: t("sync.rooms.leave"), run: leave }
				: {
						icon: LogIn,
						label: t("sync.rooms.connect"),
						run: () => join(room),
					},
		];
		if (room.id === ACTIVE_ID) {
			menu.push({
				icon: BookmarkPlus,
				label: t("sync.rooms.save"),
				run: () => persist(room),
			});
		}
		// What is shared is the line the paste detector reads back, password
		// included: whoever receives it enters without typing either.
		menu.push(
			...shareLink(roomLink(room.code, room.password), t("sync.rooms.share")),
		);
		return {
			id: room.id,
			name: room.name || room.code,
			cover,
			subtitle: song ? (
				<span class="flex flex-col">
					<span class="truncate">{song.name}</span>
					{artist && <span class="truncate">{artist}</span>}
				</span>
			) : undefined,
			placeholder: renderStatus,
			overlay: cover ? renderStatus : undefined,
			menu,
			removable: room.id !== ACTIVE_ID,
		};
	});

	const buttonClass =
		"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md bg-black/70 text-zinc-300";

	const reorder = (fromId: string, toId: string) => {
		const next = move(
			ordered.map((room) => room.id),
			fromId,
			toId,
		);
		queryClient.setQueryData(["shelf-order", "rooms"], next);
		void saveOrder("rooms", "", next);
	};

	return (
		<>
			<Shelf
				items={items}
				to={() => "/"}
				viewKey="rooms_view"
				// The hint carries the button while the shelf is empty, so this one
				// waits for rows; a session does not hide it either, because a new room
				// is another room to enter, not a switch.
				toolbar={
					<>
						{(ordered.length > 0 || rooms.isLoading) && (
							<button
								type="button"
								onClick={create}
								class="flex h-10 md:h-8 shrink-0 cursor-pointer items-center gap-1.5 rounded-md border border-zinc-800 bg-zinc-950 px-3 text-sm text-zinc-500 transition-colors hover:text-zinc-50"
							>
								<Plus size={16} class="md:hidden" />
								<Plus size={14} class="hidden md:block" />
								{t("sync.rooms.create")}
							</button>
						)}
						{role !== "off" && (
							<button
								type="button"
								onClick={leave}
								class="flex h-10 md:h-8 shrink-0 cursor-pointer items-center gap-1.5 rounded-md bg-red-900/40 px-3 text-sm text-red-200 transition-colors hover:bg-red-900/60"
							>
								<session.icon size={16} class="md:hidden" />
								<session.icon size={14} class="hidden md:block" />
								{session.label}
							</button>
						)}
					</>
				}
				onSelect={open}
				onReorder={reorder}
				actions={(id) => {
					const room = rows.find((r) => r.id === id);
					if (!room) return null;
					const isActive = room.code === activeCode;
					const connecting = isActive && connectingCode === room.code;
					const connected = isActive && connectedCode === room.code;

					if (connecting) {
						return (
							<span
								title={t("sync.rooms.status.connecting")}
								class={buttonClass}
							>
								<Loader size={16} class="animate-spin text-amber-400" />
							</span>
						);
					}

					return (
						<>
							{connected ? (
								<button
									type="button"
									title={t("sync.rooms.leave")}
									onClick={(e) => {
										e.preventDefault();
										e.stopPropagation();
										leave();
									}}
									class={cn(buttonClass, "hover:text-red-300")}
								>
									<LogOut size={16} />
								</button>
							) : (
								<button
									type="button"
									title={t("sync.rooms.connect")}
									onClick={(e) => {
										e.preventDefault();
										e.stopPropagation();
										join(room);
									}}
									class={cn(buttonClass, "hover:text-green-400")}
								>
									<LogIn size={16} />
								</button>
							)}
							{room.id === ACTIVE_ID && (
								<button
									type="button"
									title={t("sync.rooms.save")}
									onClick={(e) => {
										e.preventDefault();
										e.stopPropagation();
										persist(room);
									}}
									class={cn(buttonClass, "hover:text-amber-300")}
								>
									<BookmarkPlus size={16} />
								</button>
							)}
						</>
					);
				}}
				isLoading={rooms.isLoading}
				emptyMessage={
					relay === "" ? <RelayHint /> : <RoomsHint onNew={create} />
				}
				onRemove={onRemove}
			/>
			<ConfirmModal
				open={confirming.value !== null}
				danger
				pending={remove.isPending}
				close={() => {
					confirming.value = null;
				}}
				onConfirm={confirmRemove}
			/>
		</>
	);
}
