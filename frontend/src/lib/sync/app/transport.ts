import {
	RelayConfig,
	RelayOverride,
	SaveRelay,
	SyncCheck,
	SyncConnect,
	SyncSend,
	SyncStop,
} from "@bindings/app";
import { Events } from "@wailsio/runtime";
import { t } from "@/lib/shared/i18n";
import { translateError } from "@/lib/shared/i18n/errors";
import { set } from "@/lib/shared/utils/storage";
import { addError, logError } from "@/lib/shared/views/stores/errors";
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

// A check reports two independent things: whether the relay answered at all,
// and what it made of the token. The caller needs both, and neither is worth
// throwing, so the verdict comes back as a value with a translated reason.
export type RelayCheck = {
	error: string;
	token: "none" | "required" | "accepted" | "rejected";
};

export async function check(url: string, token: string): Promise<RelayCheck> {
	try {
		const auth = await SyncCheck(url, token);
		return { error: "", token: auth ? "accepted" : "none" };
	} catch (err) {
		// The relay is what judges the token, so its complaint describes the
		// field and not a missing relay: the URL did answer.
		if (/token/i.test(String(err))) {
			return { error: "", token: token === "" ? "required" : "rejected" };
		}
		// Whatever else went wrong (an address that is not a relay, a relay
		// that is down, a network that is gone) is the same news for the user
		// and reads as a wall of Go text, so it goes to the log instead.
		logError(err, "relay check");
		return { error: t("errors.relay.unreachable"), token: "none" };
	}
}

// save persists the relay the panel and the shelf will dial. The store is the
// home that survives a reinstall, so it is what a restart reads; localStorage
// only paints the form on the next launch without waiting for the store. A relay
// pinned by RELAY_API_URL is never written.
export function save(): void {
	if (store.relayLocked.value) return;
	const url = store.relayUrl.value.trim();
	const token = store.token.value.trim();
	set("sync:relay", url);
	set("sync:token", token);
	// A store that is down costs the user their relay on the next install, but
	// there is nothing they could do about it mid-session: log it, never show it.
	SaveRelay(url, token).catch((err) => logError(err, "relay"));
}

// loadRelay points the app at the relay it should dial: the one pinned by
// RELAY_API_URL if the environment has one, else the saved one. An install made
// before the relay moved to the store still holds it in localStorage, so a store
// with nothing in it gets seeded from the form once.
export async function loadRelay(): Promise<void> {
	const override = await RelayOverride();
	const pinned = override.url?.trim();
	if (pinned) {
		store.relayUrl.value = pinned;
		store.token.value = override.token?.trim() ?? "";
		store.relayLocked.value = true;
		return;
	}
	const saved = await RelayConfig();
	const url = saved.url?.trim() ?? "";
	if (url === "") {
		save();
		return;
	}
	store.relayUrl.value = url;
	store.token.value = saved.token?.trim() ?? "";
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
	save();
	set("sync:room", code);
	set("sync:password", store.password.value.trim());
	joinedRoom = code;
	lastRole = "";
	try {
		await dial(code, "");
	} catch (err) {
		stop();
		if (/token/i.test(String(err))) {
			store.tokenRequired.value = true;
			store.error.value = t("errors.relay.tokenRequired");
		} else {
			store.error.value = translateError(err);
		}
		addError(err, "room");
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
