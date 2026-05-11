import { post } from "@/lib/shared/api";
import { computed, signal } from "@preact/signals";

const AUTH_SYNC_BRIDGE_ID = "jota:auth-sync";

const authSyncBridge: BroadcastChannel | null =
  typeof BroadcastChannel === "undefined"
    ? null
    : new BroadcastChannel(AUTH_SYNC_BRIDGE_ID);

function propagateAuthChange(): void {
  try {
    authSyncBridge?.postMessage({ kind: "auth/changed" as const });
  } catch {
    // ignore
  }
}

if (authSyncBridge) {
  authSyncBridge.addEventListener("message", () => {
    void syncAuth();
  });
}

export const currentUser = signal<string | null>(null);

export const authKnown = signal(false);

export type AuthPhase = "loading" | "guest" | "signedIn";

export const authPhase = computed<AuthPhase>(() => {
  if (!authKnown.value) return "loading";
  return currentUser.value !== null ? "signedIn" : "guest";
});

interface MeResponse {
  user?: string;
}

export async function syncAuth(): Promise<void> {
  try {
    const url = new URL("/api/auth/me", location.origin);
    const res = await fetch(url, {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" },
    });
    if (!res.ok) {
      currentUser.value = null;
      return;
    }
    const data = (await res.json()) as MeResponse;
    currentUser.value =
      typeof data.user === "string" && data.user.length > 0 ? data.user : null;
  } catch {
    currentUser.value = null;
  } finally {
    authKnown.value = true;
  }
}

export function logIn(userName: string): void {
  currentUser.value = userName;
  authKnown.value = true;
  propagateAuthChange();
}

export async function logOut(): Promise<void> {
  try {
    await post<{ message?: string }>("/auth/log-out", {});
  } catch {
    // best-effort
  }
  await syncAuth();
  propagateAuthChange();
}
