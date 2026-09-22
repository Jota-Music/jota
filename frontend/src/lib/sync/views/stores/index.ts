import { signal } from "@preact/signals";
import { get } from "@/lib/shared/utils/storage";

type Status = "idle" | "connecting" | "open" | "closed";

export const role = signal<"off" | "host" | "guest">("off");
export const status = signal<Status>("idle");
export const peers = signal(0);
// True once the room has answered our join, so the UI can tell connecting from
// actually listening.
export const joined = signal(false);
export const room = signal(get("sync:room"));
export const relayUrl = signal(get("sync:relay"));
export const token = signal(get("sync:token"));
// True when RELAY_API_URL pins the relay from the environment: the value wins
// over the saved one and is not persisted.
export const relayLocked = signal(false);
export const tokenRequired = signal(false);
export const password = signal(get("sync:password"));
export const error = signal("");
export const offsetMs = signal(0);
export const showSync = signal(false);
