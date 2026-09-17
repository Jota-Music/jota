import { ListRooms, RemoveRoom, SaveRoom, SyncRoomStatus } from "@bindings/app";
import type { Room } from "../../../../bindings/github.com/Jota-Music/jota/internal/services/rooms/models";
import type { RoomStatus } from "../../../../bindings/github.com/Jota-Music/jota/internal/services/sync/models";

export type { Room, RoomStatus };

export async function listRooms(): Promise<Room[]> {
	return (await ListRooms()) ?? [];
}

export async function saveRoom(room: Room): Promise<Room[]> {
	return (await SaveRoom(room)) ?? [];
}

export async function removeRoom(id: string): Promise<Room[]> {
	return (await RemoveRoom(id)) ?? [];
}

export async function roomStatus(
	relayUrl: string,
	code: string,
	token: string,
): Promise<RoomStatus> {
	return await SyncRoomStatus(relayUrl, code, token);
}
