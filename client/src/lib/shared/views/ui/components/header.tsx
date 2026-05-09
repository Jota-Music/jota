import {
    authKnown,
    authPhase,
    currentUser,
    logOut,
} from "@/lib/auth/views/stores/session";
import {
    goToMyHomeRoom,
    homeRoomEnforced,
    joinRoomById,
    roomState,
} from "@/lib/shared/api/room";
import { ws } from "@/lib/shared/api/socket";
import { cn } from "@/lib/shared/utils/tw";
import {
    ArrowLeft,
    Copy,
    House,
    LockKeyhole,
    LockKeyholeOpen,
    LogIn,
    LogOut,
    Turntable,
    Undo2,
    Users,
} from "lucide-preact";
import { useCallback, useState } from "preact/hooks";
import { Link, useLocation } from "wouter-preact";

export function Header({
    class: _class,
    className,
}: {
    class?: string;
    className?: string;
}) {
    const [, setLocation] = useLocation();

    const [joinDraft, setJoinDraft] = useState("");
    const [openSettings, setOpenSettings] = useState(false);

    const onJoinSubmit = useCallback(
        (e: Event) => {
            e.preventDefault();
            if (!joinDraft) return;
            joinRoomById(joinDraft);
            setJoinDraft("");
            setOpenSettings(false);
        },
        [joinDraft],
    );

    const phase = authPhase.value;
    const user = currentUser.value;
    const known = authKnown.value;

    const { id: room, visibility: vis, guests, youAreOwner } =
        roomState.value;

    const browsing = user != null && !homeRoomEnforced.value;

    const canToggleVisibility =
        youAreOwner || (user != null && room === user);

    const copyRoomId = useCallback(() => {
        if (!room) return;
        navigator.clipboard.writeText(room);
    }, [room]);

    // ✅ BACK CORRECTO
    const goBack = useCallback(() => {
        if (window.history.length > 1) {
            window.history.back();
        } else {
            setLocation("/");
        }
    }, [setLocation]);

    return (
        <header
            class={cn(
                "sticky top-0 z-40 border-b border-zinc-800 bg-zinc-950/80 backdrop-blur",
                _class,
                className,
            )}
        >
            {/* Top bar */}
            <div class="mx-auto flex max-w-2xl px-2 md:px-0 items-center justify-between py-2 text-sm text-zinc-300">

                <div class="flex items-center gap-3">
                    {/* BACK */}
                    <button
                        type="button"
                        onClick={goBack}
                        class="text-zinc-400 hover:text-zinc-100 cursor-pointer"
                    >
                        <ArrowLeft class="size-4" />
                    </button>

                    {/* HOME */}
                    <Link
                        href="/"
                        class="text-zinc-400 hover:text-zinc-100"
                    >
                        <House class="size-4" />
                    </Link>
                </div>

                <div class="flex items-center gap-3">
                    <div class="flex items-center gap-2 text-xs text-zinc-500">
                        <span class="flex gap-0.5 items-center justify-center">
                            <Users class="size-3" />
                            {guests ?? 0}
                        </span>

                        <button
                            type="button"
                            class={cn(
                                "rounded p-1.5 flex items-center cursor-pointer",
                                vis === "private"
                                    ? "bg-amber-900/40 text-amber-200"
                                    : "bg-emerald-900/30 text-emerald-200",
                            )}
                            disabled={!canToggleVisibility}
                            onClick={() =>
                                ws.send("set-visibility", {
                                    visibility:
                                        vis === "private"
                                            ? "public"
                                            : "private",
                                })
                            }
                        >
                            {vis === "private" ? (
                                <LockKeyhole class="size-3" />
                            ) : (
                                <LockKeyholeOpen class="size-3" />
                            )}
                        </button>
                    </div>

                    <button
                        type="button"
                        onClick={() => setOpenSettings((v) => !v)}
                        class="text-zinc-400 hover:text-zinc-100 cursor-pointer"
                    >
                        <Turntable class="size-4" />
                    </button>
                </div>

                <div class="flex items-center gap-2">
                    {!known ? (
                        <span class="text-zinc-600">…</span>
                    ) : phase === "guest" ? (
                        <Link href="/register">
                            <LogIn class="size-4 text-zinc-400" />
                        </Link>
                    ) : (
                        <button
                            onClick={() => logOut()}
                            type="button"
                            class="cursor-pointer"
                        >
                            <LogOut class="size-4 text-zinc-400 hover:text-zinc-100" />
                        </button>
                    )}
                </div>
            </div>

            {/* Room form */}
            <div
                class={cn(
                    "mx-auto max-w-2xl overflow-hidden transition-all duration-200",
                    openSettings
                        ? "max-h-32 opacity-100 py-2"
                        : "max-h-0 opacity-0 py-0",
                )}
            >
                <form
                    onSubmit={onJoinSubmit}
                    class="flex flex-col md:flex-row md:items-center gap-2 px-2 md:px-0"
                >
                    <input
                        class="h-10 w-full rounded-md border border-zinc-800 bg-zinc-950 pl-3 text-sm text-white outline-none focus:border-zinc-600"
                        placeholder={room}
                        value={joinDraft}
                        onInput={(e) =>
                            setJoinDraft(
                                (e.target as HTMLInputElement).value,
                            )
                        }
                    />

                    <div class="flex items-center gap-2 w-full md:w-auto">
                        <button
                            class="flex-1 md:flex-none rounded-lg px-4 py-2 font-bold text-sm text-(--binary-color) bg-(--dominant-color) hover:opacity-75 transition-opacity cursor-pointer"
                            type="submit"
                        >
                            Join
                        </button>

                        <button
                            type="button"
                            class="shrink-0 text-zinc-500 hover:text-zinc-300 cursor-pointer p-2"
                            onClick={() => goToMyHomeRoom()}
                            disabled={!browsing && room === user}
                            title="Mi sala"
                        >
                            <Undo2 class="size-4" />
                        </button>

                        <button
                            type="button"
                            class="shrink-0 text-zinc-500 hover:text-zinc-300 cursor-pointer p-2"
                            onClick={copyRoomId}
                            title="Copiar ID"
                        >
                            <Copy class="size-4" />
                        </button>
                    </div>
                </form>
            </div>
        </header>
    );
}