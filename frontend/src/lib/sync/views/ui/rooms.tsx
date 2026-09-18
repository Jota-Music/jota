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
	PowerOff,
	TriangleAlert,
	Wifi,
	WifiOff,
} from "lucide-preact";
import { useEffect } from "preact/hooks";
import { type Item, Shelf } from "@/lib/music/views/ui/shelf";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import {
	listRooms,
	type Room,
	type RoomStatus,
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
				<Loader size={size} class="animate-spin text-amber-400" />
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
				<Loader size={size} class="animate-spin text-zinc-500" />
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

export function RoomsShelf() {
	const queryClient = useQueryClient();
	const rooms = useQuery({ queryKey: ["rooms"], queryFn: listRooms });

	const list = rooms.data ?? [];
	const locked = store.relayLocked.value;
	const status = store.status.value;
	const role = store.role.value;
	const current = store.room.value;

	const activeCode = current && status !== "idle" ? current : "";
	const connectingCode =
		status === "connecting" || status === "closed" ? current : "";
	const connectedCode = status === "open" && role !== "off" ? current : "";

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
		store.room.value = room.code;
		store.showSync.value = true;
	};

	const join = (room: Room) => {
		apply(room);
		void transport.connect(room.code);
	};

	const leave = () => {
		transport.stop();
	};

	const onRemove = (id: string) => {
		const room = rows.find((r) => r.id === id);
		// A room we are in is left first, so we never stay in a room the shelf no
		// longer shows.
		if (room && room.code === activeCode) transport.stop();
		remove.mutate(id);
	};

	const items: Item[] = rows.map((room) => {
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
		return {
			id: room.id,
			name: room.name || room.code,
			placeholder: (size) => (
				<Status
					status={query?.data}
					pending={query?.isLoading}
					size={size}
					live={live}
					members={members}
					needsPassword={!!query?.data?.locked && !room.password}
				/>
			),
			removable: room.id !== ACTIVE_ID,
		};
	});

	const buttonClass =
		"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md bg-black/70 text-zinc-300";

	return (
		<Shelf
			items={items}
			to={() => "/"}
			viewKey="rooms_view"
			onSelect={open}
			actions={(id) => {
				const room = rows.find((r) => r.id === id);
				if (!room) return null;
				const isActive = room.code === activeCode;
				const connecting = isActive && connectingCode === room.code;
				const connected = isActive && connectedCode === room.code;

				if (connecting) {
					return (
						<span title={t("sync.rooms.status.connecting")} class={buttonClass}>
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
									save.mutate({
										id: "",
										code: room.code,
										relayUrl: room.relayUrl,
										token: room.token,
										password: room.password,
										name: room.code,
										savedAt: 0,
									});
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
			emptyMessage={t("sync.rooms.empty")}
			onRemove={onRemove}
		/>
	);
}
