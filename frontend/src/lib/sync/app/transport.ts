import { SyncCheck, SyncConnect, SyncSend, SyncStop } from "@bindings/app";
import { Events } from "@wailsio/runtime";
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

let pendingSnapshot: string | null = null;
let snapshotScheduled = false;

Events.On("sync:message", (ev) => {
	const raw = String(ev.data ?? "");
	let type = "";
	try {
		type = (JSON.parse(raw) as { t?: string }).t ?? "";
	} catch {
		// not JSON, forward as-is
	}
	if (type === "state" || type === "heartbeat") {
		pendingSnapshot = raw;
		if (!snapshotScheduled) {
			snapshotScheduled = true;
			window.setTimeout(flushSnapshot, 50);
		}
		return;
	}
	store.lastMessage.value = raw;
});

function flushSnapshot(): void {
	snapshotScheduled = false;
	const raw = pendingSnapshot;
	pendingSnapshot = null;
	if (raw != null && raw !== store.lastMessage.value) {
		store.lastMessage.value = raw;
	}
}

export async function check(url: string): Promise<boolean> {
	return await SyncCheck(url);
}

export async function connect(code: string): Promise<void> {
	reset();
	const url = store.relayUrl.value.trim();
	const token = store.token.value.trim();
	const password = store.password.value.trim();
	localStorage.setItem("sync:relay", url);
	localStorage.setItem("sync:token", token);
	localStorage.setItem("sync:room", code);
	localStorage.setItem("sync:password", password);
	store.room.value = code;
	store.status.value = "connecting";
	try {
		await SyncConnect(url, code, "", token, password);
	} catch (err) {
		stop();
		const msg = String(err);
		if (/token/i.test(msg)) {
			store.tokenRequired.value = true;
			store.error.value = "This relay requires a token.";
		} else {
			store.error.value = msg;
		}
	}
}

export function stop(): void {
	SyncStop();
	reset();
}

export function send(msg: PeerMessage): void {
	SyncSend(JSON.stringify(msg)).catch(() => {});
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
