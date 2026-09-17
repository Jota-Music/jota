import { useMutation, useQueryClient } from "@tanstack/preact-query";
import { useEffect, useRef, useState } from "preact/hooks";
import {
	listRooms,
	type Room,
	removeRoom,
	saveRoom,
} from "@/lib/sync/app/rooms";
import * as transport from "@/lib/sync/app/transport";
import * as store from "@/lib/sync/views/stores";

function randomCode(): string {
	return Math.random().toString(36).slice(2, 8);
}

// A room is identified by relay and code, the same key the store uses.
function identity(relayUrl: string, code: string): string {
	return `${relayUrl.trim().toLowerCase()}|${code.trim()}`;
}

// Form state and actions for the Rooms panel: what the user typed, what is
// saved, and the connect/save/unsave/leave actions. The panel stays declarative.
export function useRoomForm() {
	const queryClient = useQueryClient();
	const [code, setCode] = useState(store.room.value);

	const role = store.role.value;
	const status = store.status.value;
	const relay = store.relayUrl.value.trim();
	const currentRoom = store.room.value.trim();
	const password = store.password.value;
	const token = store.token.value.trim();

	const ready = relay !== "" && code.trim() !== "";
	const connecting = status === "connecting";
	const active = role !== "off";
	const switching = active && code.trim() !== currentRoom;

	const [savedRooms, setSavedRooms] = useState<Room[]>(
		() => queryClient.getQueryData<Room[]>(["rooms"]) ?? [],
	);

	const savedRoom = savedRooms.find(
		(room) => identity(room.relayUrl, room.code) === identity(relay, code),
	);
	// A relay pinned by the environment must not leak its token into a saved
	// room, so an existing room keeps the token it already had.
	const savedToken =
		store.relayLocked.value && savedRoom ? savedRoom.token : token;
	const changed =
		!!savedRoom &&
		(savedRoom.password !== password.trim() || savedRoom.token !== savedToken);

	const keep = (next: Room[]) => {
		setSavedRooms(next);
		queryClient.setQueryData(["rooms"], next);
	};

	const save = useMutation({
		mutationFn: () =>
			saveRoom({
				id: "",
				code: code.trim(),
				relayUrl: relay,
				token: savedToken,
				password: password.trim(),
				name: code.trim(),
				savedAt: 0,
			}),
		onSuccess: keep,
	});
	const remove = useMutation({ mutationFn: removeRoom, onSuccess: keep });
	const commit = save.mutate;

	// The shelf owns the saved list; read it through the cache so query-core
	// stays out of the eager bundle that useQuery would pull in.
	useEffect(() => {
		let alive = true;
		void queryClient
			.fetchQuery({ queryKey: ["rooms"], queryFn: listRooms })
			.then((next) => {
				if (alive) setSavedRooms(next);
			});
		return () => {
			alive = false;
		};
	}, [queryClient]);

	// Switching rooms must not carry the previous password: take the target
	// room's saved one, or clear it when it has none.
	const lastCode = useRef(code);
	useEffect(() => {
		if (code.trim() === lastCode.current.trim()) return;
		lastCode.current = code;
		store.password.value = savedRoom?.password ?? "";
	}, [code, savedRoom]);

	// Closing the panel without a room drops a password typed for nothing; a
	// room still connected keeps its password for a reconnect.
	useEffect(() => {
		return () => {
			if (store.role.value === "off") store.password.value = "";
		};
	}, []);

	// Editing a saved room updates it after a pause; leaving the field commits
	// right away. `password` and `savedToken` are deps so each keystroke restarts
	// the pause instead of firing on the first one.
	useEffect(() => {
		if (!savedRoom) return;
		if (
			savedRoom.password === password.trim() &&
			savedRoom.token === savedToken
		) {
			return;
		}
		const id = setTimeout(commit, 600);
		return () => clearTimeout(id);
	}, [savedRoom, password, savedToken, commit]);

	return {
		code,
		setCode,
		generate: () => setCode(randomCode()),

		password,
		setPassword: (value: string) => {
			store.password.value = value;
		},
		commitPassword: () => {
			if (changed) commit();
		},

		relay,
		ready,
		connecting,
		active,
		currentRoom,
		switching,
		savedRoom,
		host: role === "host",

		saving: save.isPending,
		removing: remove.isPending,

		connect: () => void transport.connect(code.trim()),
		leave: () => transport.stop(),
		save: () => commit(),
		unsave: (id: string) => remove.mutate(id),
	};
}
