import { signal } from "@preact/signals";

export type Status = "idle" | "connecting" | "open" | "closed";

export const role = signal<"off" | "host" | "guest">("off");
export const status = signal<Status>("idle");
export const peers = signal(0);
export const room = signal("");
export const relayUrl = signal(localStorage.getItem("sync:relay") ?? "");
export const error = signal("");
export const offsetMs = signal(0);
export const lastMessage = signal("");
export const showSync = signal(false);
