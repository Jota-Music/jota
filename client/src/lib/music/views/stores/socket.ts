import type { Song } from "@/lib/music/model";
import { remoteTogglePlayPause } from "@/lib/music/views/stores/audio";
import {
	applyRemoteSeek,
	applyRoomPlaybackFromPeer,
	handlePlay,
	handleRemoteNewTrack,
} from "@/lib/music/views/stores/player";
import { addError } from "@/lib/shared/views/stores/errors";
import {
	playerSkeletonOn,
	shareSnapshot,
} from "@/lib/music/views/stores/queue";
import type { RoomState } from "@/lib/shared/api/room";
import {
	patchRoom,
	roomState,
	rotateGuestRoomIfForbidden,
} from "@/lib/shared/api/room";
import { ws } from "@/lib/shared/api/socket";

let prevRoomId = roomState.peek().id;
roomState.subscribe(() => {
	const id = roomState.value.id;
	if (id === prevRoomId) return;
	prevRoomId = id;
	playerSkeletonOn.value = true;
});

ws.on("room-forbidden", () => {
	addError("Room is no longer accessible");
	rotateGuestRoomIfForbidden();
});

ws.on("join-denied", () => {
	addError("Could not join room — access denied");
	rotateGuestRoomIfForbidden();
});

ws.on("kicked", () => {
	addError("You have been removed from the room");
	rotateGuestRoomIfForbidden();
});

ws.on(
	"joined",
	({
		data,
	}: {
		data: {
			youAreOwner?: boolean;
			visibility?: string;
			headcount?: number;
			awaitingSnapshot?: boolean;
		};
	}) => {
		const patch: Partial<Omit<RoomState, "id">> = {
			youAreOwner: !!data?.youAreOwner,
		};
		const v = data?.visibility;
		if (v === "public" || v === "private") {
			patch.visibility = v;
		}
		const n = data?.headcount;
		if (typeof n === "number" && Number.isFinite(n) && n >= 0) {
			patch.guests = n;
		}
		patchRoom(patch);
		if (data?.youAreOwner) shareSnapshot();
		playerSkeletonOn.value = !(
			data?.youAreOwner || data?.awaitingSnapshot === false
		);
	},
);

ws.on("room-count", ({ data }: { data: { count?: number } }) => {
	const n = data?.count;
	if (typeof n === "number" && Number.isFinite(n) && n >= 0) {
		patchRoom({ guests: n });
	}
});

ws.on("visibility", ({ data }: { data: { visibility?: string } }) => {
	const v = data?.visibility;
	if (v === "public" || v === "private") {
		patchRoom({ visibility: v });
	}

	console.log("Room visibility is now", v);
});

ws.on("request-snapshot", () => {
	shareSnapshot();
});

ws.on("toggle", async () => {
	await remoteTogglePlayPause();
});

interface SnapshotPayload {
	queue: Song[];
	index: number;
	playing: boolean;
	position?: number;
}

interface SnapshotMessage {
	data: SnapshotPayload;
}

ws.on("snapshot", (message: SnapshotMessage) => {
	const { queue: q, index, playing, position } = message.data;
	playerSkeletonOn.value = false;
	void applyRoomPlaybackFromPeer(q, index, playing, position);
});

interface SeekMessage {
	data: { position: number };
}

ws.on("seek", (message: SeekMessage) => {
	const p = message.data?.position;
	if (typeof p !== "number" || !Number.isFinite(p)) return;
	void applyRemoteSeek(p);
});

interface NewTrackPayload {
	queue: Song[];
	index: number;
	playing: boolean;
	position?: number;
	generation?: string;
}

interface NewTrackMessage {
	data: NewTrackPayload;
}

ws.on("new-track", (message: NewTrackMessage) => {
	const { queue: q, index, playing, position, generation } = message.data;
	playerSkeletonOn.value = false;
	void handleRemoteNewTrack(q, index, playing, position, generation);
});

ws.on("play", () => {
	handlePlay();
});

ws.on("error", ({ data }) => {
	console.error("Error from server:", data);
});
