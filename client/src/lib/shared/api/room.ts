import { effect, signal } from "@preact/signals";
import { authKnown, currentUser } from "@/lib/auth/views/stores/session";

export const generateGuestRoomId = () => crypto.randomUUID();

const LEGACY_PLACEHOLDER_ROOM = "tifiem";

function readInitialRoomId(): string {
	if (typeof localStorage === "undefined") {
		return generateGuestRoomId();
	}
	const stored = localStorage.getItem("room");
	if (stored == null || stored === "" || stored === LEGACY_PLACEHOLDER_ROOM) {
		return generateGuestRoomId();
	}
	return stored;
}

export type RoomVisibility = "public" | "private";

export type RoomState = {
	id: string;
	visibility: RoomVisibility;
	guests: number;
	youAreOwner: boolean;
};

function freshRoomState(id: string): RoomState {
	return {
		id,
		visibility: "public",
		guests: 0,
		youAreOwner: false,
	};
}

export const roomState = signal<RoomState>(freshRoomState(readInitialRoomId()));

export function setRoomId(id: string): void {
	if (id === roomState.value.id) return;
	roomState.value = freshRoomState(id);
}

export function patchRoom(p: Partial<Omit<RoomState, "id">>): void {
	roomState.value = { ...roomState.value, ...p };
}

export const homeRoomEnforced = signal(true);

effect(() => {
	const id = roomState.value.id;
	if (typeof localStorage === "undefined") return;
	try {
		localStorage.setItem("room", id);
	} catch {
		/* ignore quota / private mode */
	}
});

let wasSignedIn = false;

effect(() => {
	void authKnown.value;
	void currentUser.value;
	void homeRoomEnforced.value;

	if (!authKnown.value) {
		return;
	}

	const user = currentUser.value;
	if (user) {
		if (homeRoomEnforced.value && roomState.value.id !== user) {
			setRoomId(user);
		}
		wasSignedIn = true;
	} else {
		if (wasSignedIn) {
			setRoomId(generateGuestRoomId());
		}
		wasSignedIn = false;
		homeRoomEnforced.value = true;
	}
});

export function joinRoomById(roomId: string): void {
	let id = roomId.trim();
	if (!id) return;
	if (id === LEGACY_PLACEHOLDER_ROOM) {
		id = generateGuestRoomId();
	}
	if (id === roomState.value.id) return;
	if (currentUser.value) {
		homeRoomEnforced.value = false;
	}
	setRoomId(id);
}

export function goToMyHomeRoom(): void {
	const u = currentUser.value;
	if (!u) return;
	homeRoomEnforced.value = true;
	setRoomId(u);
}

export function rotateGuestRoomIfForbidden(): void {
	if (!authKnown.value) {
		return;
	}
	const user = currentUser.value;
	if (user != null && roomState.value.id === user) {
		return;
	}
	setRoomId(generateGuestRoomId());
}
