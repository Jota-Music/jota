import {
    authKnown,
    authPhase,
    currentUser,
    logOut,
    syncAuth,
} from "@/lib/auth/views/stores/session";

export function SessionDebugNav() {
    if (!import.meta.env.DEV) {
        return null;
    }

    const phase = authPhase.value;
    const user = currentUser.value;
    const known = authKnown.value;

    return (
        <nav
            class="pointer-events-auto fixed bottom-0 left-0 right-0 z-[100] border-t border-zinc-700 bg-zinc-900/95 px-2 py-1.5 text-[11px] leading-tight font-mono text-zinc-100 backdrop-blur-sm"
            aria-label="Depuración de sesión"
        >
            <div class="mx-auto flex max-w-2xl flex-wrap items-center gap-x-3 gap-y-1">
                <span class="text-zinc-500">auth</span>
                <span
                    class={
                        phase === "signedIn"
                            ? "text-emerald-400"
                            : phase === "loading"
                              ? "text-amber-400"
                              : "text-zinc-300"
                    }
                >
                    {phase}
                </span>
                <span class="text-zinc-500">
                    known:{known ? "y" : "n"}
                </span>
                {user != null && user !== "" ? (
                    <span class="truncate text-zinc-200" title={user}>
                        user:{user}
                    </span>
                ) : (
                    <span class="text-zinc-500">user:—</span>
                )}
                <span class="ml-auto flex shrink-0 gap-1">
                    <button
                        type="button"
                        class="rounded border border-zinc-600 px-1.5 py-0.5 text-zinc-300 hover:bg-zinc-800"
                        onClick={() => void syncAuth()}
                    >
                        sync
                    </button>
                    <button
                        type="button"
                        class="rounded border border-zinc-600 px-1.5 py-0.5 text-zinc-300 hover:bg-zinc-800"
                        onClick={() => void logOut()}
                    >
                        log out
                    </button>
                </span>
            </div>
        </nav>
    );
}
