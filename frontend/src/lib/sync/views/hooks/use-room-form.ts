import { useMutation, useQueryClient } from "@tanstack/preact-query";
import { useEffect, useRef, useState } from "preact/hooks";
import {
	listRooms,
	type Room,
	randomCode,
	saveRoom,
} from "@/lib/sync/app/rooms";
import * as transport from "@/lib/sync/app/transport";
import * as store from "@/lib/sync/views/stores";

// A room is identified by relay and code, the same key the store uses.
function identity(relayUrl: string, code: string): string {
	return `${relayUrl.trim().toLowerCase()}|${code.trim()}`;
}

// Form state and actions for the Rooms panel: what the user typed, what is
// saved, and the save action. The panel stays declarative: naming a room and
// entering one are two different gestures, and only the shelf enters.
export function useRoomForm() {
	const queryClient = useQueryClient();
	const [code, setCode] = useState(() => store.takeDraft() ?? store.room.value);

	const relay = store.relayUrl.value.trim();
	const token = store.token.value.trim();
	// The password is the panel's own field, not the session's, until the user
	// asks to join: the room we are connected to never lends its secret to the
	// one being named here, and a room that is only saved never dials.
	const [password, setPassword] = useState("");

	const ready = relay !== "" && code.trim() !== "";

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

	// A code change brings that room's saved password, and clears it when the room
	// has none: what was typed for another room never travels. The first pass runs
	// on mount, so a panel opened on a new code starts empty even while a locked
	// room is playing.
	const lastCode = useRef<string | null>(null);
	useEffect(() => {
		if (lastCode.current === code.trim()) return;
		lastCode.current = code.trim();
		setPassword(savedRoom?.password ?? "");
	}, [code, savedRoom]);

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

	// The panel's password is what the session dials: joining spends it on the
	// store, saving never does.
	const join = () => {
		if (!ready) return;
		store.password.value = password.trim();
		void transport.connect(code.trim());
	};

	return {
		code,
		setCode,
		generate: () => setCode(randomCode()),

		password,
		setPassword,
		commitPassword: () => {
			if (changed) commit();
		},

		relay,
		ready,
		savedRoom,

		saving: save.isPending,

		save: () => commit(),
		join,
	};
}
