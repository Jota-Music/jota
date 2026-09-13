import {
	ReadClipboard,
	SyncCheck,
	SyncConnect,
	SyncSend,
	SyncStop,
} from "@bindings/app";
import { Clipboard, Events } from "@wailsio/runtime";
import type { PeerMessage } from "@/lib/sync/model";
import * as store from "@/lib/sync/views/stores";

Events.On("sync:connected", () => {
	store.status.value = "open";
});

Events.On("sync:closed", () => {
	store.status.value = "closed";
	store.role.value = "off";
	store.peers.value = 0;
});

Events.On("sync:error", (ev) => {
	store.error.value = String(ev.data ?? "sync error");
});

Events.On("sync:message", (ev) => {
	store.lastMessage.value = String(ev.data ?? "");
});

export async function check(url: string): Promise<void> {
	await SyncCheck(url);
}

export async function connect(code: string): Promise<void> {
	reset();
	const url = store.relayUrl.value.trim();
	localStorage.setItem("sync:relay", url);
	store.room.value = code;
	store.status.value = "connecting";
	try {
		await SyncConnect(url, code, "");
	} catch (err) {
		store.status.value = "closed";
		store.role.value = "off";
		store.error.value = String(err);
	}
}

export function stop(): void {
	SyncStop();
	reset();
}

export function send(msg: PeerMessage): void {
	SyncSend(JSON.stringify(msg)).catch(() => {});
}

export async function paste(): Promise<string> {
	try {
		return await ReadClipboard();
	} catch {
		return "";
	}
}

export async function copy(text: string): Promise<void> {
	await Clipboard.SetText(text);
}

function reset(): void {
	store.role.value = "off";
	store.status.value = "idle";
	store.room.value = "";
	store.error.value = "";
	store.lastMessage.value = "";
	store.offsetMs.value = 0;
	store.peers.value = 0;
}
