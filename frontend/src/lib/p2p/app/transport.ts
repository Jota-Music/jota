import {
	P2PGuestJoin,
	P2PHostAccept,
	P2PHostStart,
	P2PReadClipboard,
	P2PSend,
	P2PStop,
} from "@bindings/app";
import { Clipboard, Events } from "@wailsio/runtime";
import type { PeerMessage } from "@/lib/p2p/model";
import * as store from "@/lib/p2p/views/stores";

Events.On("p2p:connected", () => {
	store.status.value = "open";
});

Events.On("p2p:closed", () => {
	store.status.value = "closed";
	store.role.value = "off";
	store.inviteCode.value = "";
	store.answerCode.value = "";
});

Events.On("p2p:error", (ev) => {
	store.error.value = String(ev.data ?? "p2p error");
});

Events.On("p2p:message", (ev) => {
	store.lastMessage.value = String(ev.data ?? "");
});

export async function startHost(): Promise<string> {
	reset();
	store.role.value = "host";
	store.status.value = "connecting";
	const code = await P2PHostStart();
	store.inviteCode.value = code;
	return code;
}

export async function joinGuest(offer: string): Promise<string> {
	reset();
	store.role.value = "guest";
	store.status.value = "connecting";
	const code = await P2PGuestJoin(offer);
	store.answerCode.value = code;
	return code;
}

export async function acceptAnswer(answer: string): Promise<void> {
	await P2PHostAccept(answer);
}

export function stop(): void {
	P2PStop();
	reset();
}

export function send(msg: PeerMessage): void {
	P2PSend(JSON.stringify(msg)).catch(() => {});
}

export async function copy(text: string): Promise<void> {
	await Clipboard.SetText(text);
}

export async function paste(): Promise<string> {
	try {
		return await P2PReadClipboard();
	} catch {
		return "";
	}
}

function reset(): void {
	store.role.value = "off";
	store.status.value = "idle";
	store.inviteCode.value = "";
	store.answerCode.value = "";
	store.error.value = "";
	store.lastMessage.value = "";
	store.offsetMs.value = 0;
	store.qrValue.value = null;
}
