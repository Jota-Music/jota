import { signal } from "@preact/signals";

type Status = "idle" | "connecting" | "open" | "closed";

export const role = signal<"off" | "host" | "guest">("off");
export const status = signal<Status>("idle");
export const peers = signal(0);
// True once the room has answered our join, so the UI can tell connecting from
// actually listening.
export const joined = signal(false);
export const room = signal(localStorage.getItem("sync:room") ?? "");
export const relayUrl = signal(localStorage.getItem("sync:relay") ?? "");
export const token = signal(localStorage.getItem("sync:token") ?? "");
export const tokenRequired = signal(false);
export const password = signal(localStorage.getItem("sync:password") ?? "");
export const error = signal("");
export const offsetMs = signal(0);
export const showSync = signal(false);
