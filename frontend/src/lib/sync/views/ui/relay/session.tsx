import { Wifi } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import * as store from "@/lib/sync/views/stores";
import { showSync } from "@/lib/sync/views/stores";

// Reaching a relay and being in a room are two different things, and only the
// settings screen answers the first while the Rooms panel is closed.
export function Session() {
	const active = store.room.value !== "" && store.status.value !== "idle";
	if (!active) return null;
	const connected = store.status.value === "open";

	return (
		<button
			type="button"
			onClick={() => (showSync.value = true)}
			class={cn(
				"flex w-full cursor-pointer items-center gap-2 rounded-lg border border-zinc-800 px-3 py-2 text-xs transition-colors hover:border-zinc-700",
				connected ? "text-green-400" : "text-amber-400",
			)}
		>
			<Wifi size={14} class="shrink-0" />
			<span class="min-w-0 flex-1 truncate text-left">
				{t("sync.session.inRoom", { room: store.room.value })}
			</span>
			<span class="shrink-0 tabular-nums">
				{t("sync.rooms.status.connected", { count: store.peers.value })}
			</span>
		</button>
	);
}
