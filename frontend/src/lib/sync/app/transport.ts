import {
	RelayOverride,
	SyncCheck,
	SyncConnect,
	SyncSend,
	SyncStop,
} from "@bindings/app";
import { Events } from "@wailsio/runtime";
import { t } from "@/lib/shared/i18n";
import { translateError } from "@/lib/shared/i18n/errors";
import { addError } from "@/lib/shared/views/stores/errors";
import type { ClientMessage, ServerMessage } from "@/lib/sync/model";
import * as store from "@/lib/sync/views/stores";

type Handler = (msg: ServerMessage) => void;
type VoidHandler = () => void;

const messageHandlers = new Set<Handler>();
const openHandlers = new Set<VoidHandler>();
const closeHandlers = new Set<VoidHandler>();

// Auto-reconnect state. A room that closes unexpectedly is retried with
// backoff; a room we left on purpose is not.
let joinedRoom = "";
let lastRole: "host" | "guest" | "" = "";
let intentional = false;
let reconnectTimer: number | null = null;
let reconnectDelay = 0;

function clearReconnect(): void {
	if (reconnectTimer != null) {
		clearTimeout(reconnectTimer);
		reconnectTimer = null;
	}
}

function scheduleReconnect(): void {
	if (intentional || joinedRoom === "" || reconnectTimer != null) return;
	reconnectDelay =
		reconnectDelay === 0 ? 1000 : Math.min(reconnectDelay * 2, 15000);
	const delay = reconnectDelay + Math.random() * 500;
	reconnectTimer = window.setTimeout(async () => {
		reconnectTimer = null;
		try {
			store.status.value = "connecting";
			await dial(joinedRoom, lastRole);
			reconnectDelay = 0;
		} catch {
			scheduleReconnect();
		}
	}, delay);
}

// Deliver every frame in order. Routing them through a single signal (as the old
// code did) silently dropped any frame that arrived in the same tick as another.
export function onMessage(fn: Handler): void {
	messageHandlers.add(fn);
}

// onOpen runs once per successful connection, so the room can ask for the
// snapshot again after a reconnect.
export function onOpen(fn: VoidHandler): void {
	openHandlers.add(fn);
}

export function onClosed(fn: VoidHandler): void {
	closeHandlers.add(fn);
}

Events.On("sync:connected", () => {
	store.status.value = "open";
	for (const fn of openHandlers) fn();
});

Events.On("sync:closed", () => {
	// A close we asked for (leaving, or a fatal room error) is terminal: going
	// back to "idle" clears the connecting UI instead of leaving it spinning.
	// An unexpected close keeps "closed" and is retried.
	store.status.value = intentional ? "idle" : "closed";
	// Remember the role we had so a reconnect reclaims it instead of being
	// demoted (a host that comes back as guest can no longer answer joins).
	lastRole = store.role.value === "off" ? "" : store.role.value;
	store.role.value = "off";
	store.peers.value = 0;
	for (const fn of closeHandlers) fn();
	if (!intentional) scheduleReconnect();
});

Events.On("sync:message", (ev) => {
	let msg: ServerMessage;
	try {
		msg = JSON.parse(String(ev.data ?? "")) as ServerMessage;
	} catch {
		console.error("sync: invalid message");
		return;
	}
	for (const fn of messageHandlers) fn(msg);
});

export async function check(url: string): Promise<boolean> {
	return await SyncCheck(url);
}

// applyRelayOverride points the app at the relay pinned by RELAY_API_URL, so a
// dev build can test against a local relay instead of the saved one.
export async function applyRelayOverride(): Promise<void> {
	const override = await RelayOverride();
	const url = override.url?.trim();
	if (!url) return;
	store.relayUrl.value = url;
	store.token.value = override.token?.trim() ?? "";
	store.relayLocked.value = true;
}

async function dial(code: string, role: "host" | "guest" | ""): Promise<void> {
	store.room.value = code;
	store.status.value = "connecting";
	// The relay only announces a role when it assigns one, so a reconnect that
	// asks for an explicit role must restore it locally to keep handling frames.
	if (role !== "") store.role.value = role;
	await SyncConnect(
		store.relayUrl.value.trim(),
		code,
		role,
		store.token.value.trim(),
		store.password.value.trim(),
	);
}

export async function connect(code: string): Promise<void> {
	intentional = false;
	clearReconnect();
	reconnectDelay = 0;
	reset();
	if (!store.relayLocked.value) {
		localStorage.setItem("sync:relay", store.relayUrl.value.trim());
		localStorage.setItem("sync:token", store.token.value.trim());
	}
	localStorage.setItem("sync:room", code);
	localStorage.setItem("sync:password", store.password.value.trim());
	joinedRoom = code;
	lastRole = "";
	try {
		await dial(code, "");
	} catch (err) {
		stop();
		const msg = String(err);
		if (/token/i.test(msg)) {
			store.tokenRequired.value = true;
			store.error.value = t("errors.relay.tokenRequired");
		} else {
			store.error.value = translateError(msg);
		}
		addError(store.error.value, "room");
	}
}

export function stop(): void {
	intentional = true;
	clearReconnect();
	joinedRoom = "";
	lastRole = "";
	SyncStop();
	reset();
}

// abandon gives up on the room without wiping the reported error: a fatal room
// error (host left, room full, bad password) must not trigger a reconnect that
// would silently recreate the room.
export function abandon(): void {
	intentional = true;
	clearReconnect();
	joinedRoom = "";
	lastRole = "";
}

export function send(msg: ClientMessage): void {
	SyncSend(JSON.stringify(msg)).catch(() => {});
}

function reset(): void {
	store.role.value = "off";
	store.status.value = "idle";
	store.room.value = "";
	store.error.value = "";
	store.offsetMs.value = 0;
	store.peers.value = 0;
}
